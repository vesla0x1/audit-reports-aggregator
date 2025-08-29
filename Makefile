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
	@echo "  make local-reset       - Reset local infrastructure (clean start)"
	@echo "  make terraform-init    - Initialize Terraform for LocalStack"
	@echo "  make terraform-apply   - Apply Terraform configuration to LocalStack"
	@echo "  make terraform-destroy - Destroy Terraform resources in LocalStack"
	@echo "  make build-lambda      - Build Lambda deployment packages"
	@echo "  make deploy-lambda     - Deploy Lambda functions to LocalStack"
	@echo "  make logs              - View LocalStack logs"
	@echo "  make clean             - Clean all generated files and containers"

# =====================================
# Environment Variables (from .env file or defaults)
# =====================================
# Core Configuration
ENVIRONMENT ?= local
PROJECT_NAME ?= audit-reports-aggregator
LOG_LEVEL ?= info

# AWS Configuration
AWS_REGION ?= us-east-1
AWS_ACCESS_KEY_ID ?= test
AWS_SECRET_ACCESS_KEY ?= test
LOCALSTACK_ENDPOINT ?= http://$(LOCALSTACK_HOST)

# Lambda Configuration
LAMBDA_FUNCTION_NAME_DOWNLOADER ?= audit-reports-local-downloader
LAMBDA_TIMEOUT ?= 180
LAMBDA_MEMORY_SIZE ?= 512
LAMBDA_ROLE_ARN ?= arn:aws:iam::000000000000:role/lambda-role

# Storage Configuration
S3_BUCKET ?= audit-reports-local-downloads
S3_LAMBDA_BUCKET ?= audit-reports-local-downloads-lambda

# Docker Configuration
DOCKER_NETWORK ?= audit-network
ifeq ($(ENVIRONMENT),local)
    DOCKER_COMPOSE_ENV_FILE := .env.local
    DOCKER_COMPOSE_FILE := docker-compose.local.yml
else ifeq ($(ENVIRONMENT),production)
    DOCKER_COMPOSE_ENV_FILE := .env
    DOCKER_COMPOSE_FILE := docker-compose.yml
endif

# Terraform Configuration
TF_DIR ?= terraform/$(ENVIRONMENT)

# Docker Compose commands
.PHONY: local-up
local-up:
	@echo "Starting local infrastructure..."
	@echo "Creating Docker network if needed..."
	@docker network create $(DOCKER_NETWORK) 2>/dev/null || true
	docker-compose --env-file $(DOCKER_COMPOSE_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) up -d

.PHONY: local-down
local-down:
	@echo "Stopping local infrastructure..."
	docker-compose --env-file $(DOCKER_COMPOSE_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) down

.PHONY: logs
logs:
	docker-compose --env-file $(DOCKER_COMPOSE_ENV_FILE) -f $(DOCKER_COMPOSE_FILE) logs -f localstack

# Terraform commands
.PHONY: terraform-init
terraform-init:
	@echo "Initializing Terraform..."
	cd terraform/local && terraform init;

.PHONY: terraform-apply
terraform-apply:
	@echo "Applying Terraform configuration..."
	@echo "Creating placeholder Lambda package..."
	@echo "placeholder" > "placeholder.txt"
	zip -r placeholder.zip placeholder.txt && rm placeholder.txt
	cd terraform/local && mv ../../placeholder.zip . && terraform apply -auto-approve
	rm placeholder.zip

.PHONY: terraform-destroy
terraform-destroy:
	@echo "Destroying Terraform resources..."
	cd terraform/local && terraform destroy -auto-approve

.PHONY: terraform-output
terraform-output:
	cd terraform/local && terraform output -json; \


# Lambda build and deployment
.PHONY: build-lambda
build-lambda: build-downloader build-scraper build-processor

.PHONY: build-downloader
build-downloader:
	@echo "Building downloader Lambda..."
	cd workers/downloader && \
		GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap cmd/main.go && \
		zip -j downloader.zip bootstrap && \
		rm bootstrap
	@echo "Downloader Lambda package created: workers/downloader/downloader.zip"

.PHONY: build-scraper
build-scraper:
	@echo "Building scraper Lambda..."
	@# Add scraper build commands when ready

.PHONY: build-processor
build-processor:
	@echo "Building processor Lambda..."
	@# Add processor build commands when ready

.PHONY: deploy-lambda
deploy-lambda: build-lambda
	@echo "Deploying Lambda functions to LocalStack..."
	@echo "Uploading downloader package..."
	AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
	aws --endpoint-url=$(LOCALSTACK_ENDPOINT) \
		s3 cp workers/downloader/downloader.zip \
		s3://$(S3_LAMBDA_BUCKET)/downloader.zip
	
	@echo "Updating downloader function..."
	@AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) AWS_DEFAULT_REGION=$(AWS_REGION) \
	aws --endpoint-url=$(LOCALSTACK_ENDPOINT) \
		lambda update-function-code \
		--function-name $(LAMBDA_FUNCTION_NAME_DOWNLOADER) \
		--s3-bucket $(S3_LAMBDA_BUCKET) \
		--s3-key downloader.zip || \
	( \
		echo "Update failed, trying to create function..." && \
	    AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) AWS_DEFAULT_REGION=$(AWS_REGION) \
	    aws --endpoint-url=$(LOCALSTACK_ENDPOINT) \
		lambda create-function \
		--function-name $(LAMBDA_FUNCTION_NAME_DOWNLOADER) \
		--runtime provided.al2 \
		--role $(LAMBDA_ROLE_ARN) \
		--handler bootstrap \
		--code S3Bucket=$(S3_LAMBDA_BUCKET),S3Key=downloader.zip \
		--timeout $(LAMBDA_TIMEOUT) \
		--memory-size $(LAMBDA_MEMORY_SIZE) \
		--environment Variables="{ENVIRONMENT=$(ENVIRONMENT),S3_BUCKET=$(S3_BUCKET),LOG_LEVEL=$(LOG_LEVEL)}" \
	)
	
	@echo "Lambda deployment complete!"

.PHONE: set-lambda-env
set-lambda-env:
	AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) AWS_DEFAULT_REGION=$(AWS_REGION) \
	aws --endpoint-url=$(LOCALSTACK_ENDPOINT) \
	lambda update-function-configuration \
	--function-name $(LAMBDA_FUNCTION_NAME_DOWNLOADER) \
	--environment Variables='{S3_BUCKET=audit-reports-local-reports}'

# Clean commands
.PHONY: clean
clean:
	@echo "Cleaning up..."
	docker-compose -f docker-compose.local.yml down -v
	rm -rf tmp/
	rm -rf terraform/local/.terraform
	rm -rf terraform/local/*.tfstate*
	rm -f terraform/local/placeholder.zip
	rm -f workers/downloader/downloader.zip
	rm -f workers/scraper/scraper.zip
	rm -f workers/processor/processor.zip
	@echo "Clean complete!"

# Development helpers
.PHONY: aws-local
aws-local:
	@echo "Configure AWS CLI for LocalStack:"
	@echo "export AWS_ENDPOINT_URL=$(LOCALSTACK_ENDPOINT)"
	@echo "export AWS_ACCESS_KEY_ID=test"
	@echo "export AWS_SECRET_ACCESS_KEY=test"
	@echo "export AWS_DEFAULT_REGION=$(AWS_REGION)"

.PHONY: list-resources
list-resources:
	@echo "=== S3 Buckets ==="
	@aws --endpoint-url=$(LOCALSTACK_ENDPOINT) s3 ls
	@echo ""
	@echo "=== SQS Queues ==="
	@aws --endpoint-url=$(LOCALSTACK_ENDPOINT) sqs list-queues
	@echo ""
	@echo "=== Lambda Functions ==="
	@aws --endpoint-url=$(LOCALSTACK_ENDPOINT) lambda list-functions --query 'Functions[].FunctionName'

# Watch logs
.PHONY: watch-logs
watch-logs:
	@echo "Watching Lambda logs..."
	aws --endpoint-url=$(LOCALSTACK_ENDPOINT) logs tail \
		/aws/lambda/$(PROJECT_NAME)-$(ENVIRONMENT)-downloader \
		--follow
