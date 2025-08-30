# Load environment variables from .env file if it exists
-include .env.local
-include .env

# Export all variables to sub-make processes
export

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make local-up          - Start local infrastructure with LocalStack"
	@echo "  make local-down        - Stop local infrastructure"
	@echo "  make terraform-init    - Initialize Terraform for LocalStack"
	@echo "  make terraform-apply   - Apply Terraform configuration to LocalStack"
	@echo "  make terraform-destroy - Destroy Terraform resources in LocalStack"
	@echo "  make clean             - Clean all generated files and containers"
	@echo ""
	@echo "Worker commands (run from root):"
	@echo "  make downloader-build  - Build downloader Lambda"
	@echo "  make downloader-deploy - Deploy downloader Lambda"
	@echo "  make downloader-test   - Test downloader"

# Environment Variables
ENVIRONMENT ?= local
DOCKER_NETWORK ?= audit-network
ifeq ($(ENVIRONMENT),local)
    DOCKER_COMPOSE_ENV_FILE := .env.local
    DOCKER_COMPOSE_FILE := docker-compose.local.yml
else ifeq ($(ENVIRONMENT),production)
    DOCKER_COMPOSE_ENV_FILE := .env
    DOCKER_COMPOSE_FILE := docker-compose.yml
endif

# Docker Compose commands
.PHONY: local-up
local-up:
	@echo "Starting local infrastructure..."
	@docker network create $(DOCKER_NETWORK) 2>/dev/null || true
	docker-compose --env-file $(DOCKER_COMPOSE_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) up -d

.PHONY: local-down
local-down:
	@echo "Stopping local infrastructure..."
	docker-compose --env-file $(DOCKER_COMPOSE_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) down

# Terraform commands
.PHONY: terraform-init
terraform-init:
	@echo "Initializing Terraform..."
	cd terraform/local && terraform init

.PHONY: terraform-apply
terraform-apply:
	@echo "Applying Terraform configuration..."
	cd terraform/local && terraform apply -auto-approve

.PHONY: terraform-destroy
terraform-destroy:
	@echo "Destroying Terraform resources..."
	cd terraform/local && terraform destroy -auto-approve

# Worker delegation commands
.PHONY: downloader-build
downloader-build:
	$(MAKE) -C workers/downloader lambda-build

.PHONY: downloader-deploy
downloader-deploy:
	$(MAKE) -C workers/downloader lambda-deploy

.PHONY: downloader-test
downloader-test:
	$(MAKE) -C workers/downloader test

.PHONY: downloader-env
downloader-env:
	$(MAKE) -C workers/downloader lambda-env

# Database commands remain here as they're global
.PHONY: db-migrate
db-migrate:
	@echo "Running database migrations..."
	@docker run --rm -v $(PWD)/migrations:/migrations --network host migrate/migrate \
		-path=/migrations/ -database "$(DB_URL)" up

# Clean commands
.PHONY: clean
clean:
	@echo "Cleaning up..."
	docker-compose -f docker-compose.local.yml down -v
	rm -rf tmp/
	rm -rf terraform/local/.terraform
	rm -rf terraform/local/*.tfstate*
	$(MAKE) -C workers/downloader clean