# Load parent environment variables
-include .env.local
-include .env

# Project specific variables
BINARY_NAME=downloader
DOCKER_IMAGE=audit-downloader
DOCKER_TAG=latest
LAMBDA_PACKAGE=downloader.zip

# AWS/Lambda Configuration
AWS_REGION ?= us-east-1
AWS_ACCESS_KEY_ID ?= test
AWS_SECRET_ACCESS_KEY ?= test
LOCALSTACK_ENDPOINT ?= http://localhost:4566
LAMBDA_FUNCTION_NAME ?= audit-reports-local-downloader
LAMBDA_TIMEOUT ?= 180
LAMBDA_MEMORY_SIZE ?= 512
LAMBDA_ROLE_ARN ?= arn:aws:iam::000000000000:role/lambda-role
S3_LAMBDA_BUCKET ?= audit-reports-local-lambda-deployments

.PHONY: all build test clean run

all: build

# Local development commands
build:
	go build -o bin/$(BINARY_NAME) cmd/main.go

run:
	go run cmd/main.go

test:
	go test -v -cover ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f $(LAMBDA_PACKAGE)
	rm -f bootstrap

# Lambda specific commands
.PHONY: lambda-build
lambda-build:
	@echo "Building Lambda package for $(BINARY_NAME)..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap cmd/main.go
	zip -j $(LAMBDA_PACKAGE) bootstrap
	rm bootstrap
	@echo "Lambda package created: $(LAMBDA_PACKAGE)"

.PHONY: lambda-deploy
lambda-deploy: lambda-build
	@echo "Deploying $(BINARY_NAME) Lambda to LocalStack..."
	@echo "Uploading package to S3..."
	@AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
	aws --endpoint-url=$(LOCALSTACK_ENDPOINT) \
		s3 cp $(LAMBDA_PACKAGE) s3://$(S3_LAMBDA_BUCKET)/$(LAMBDA_PACKAGE)
	
	@echo "Updating Lambda function..."
	@ENV_VARS=$$(make -s generate-env-string); \
	AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) AWS_DEFAULT_REGION=$(AWS_REGION) \
	aws --endpoint-url=$(LOCALSTACK_ENDPOINT) \
		lambda update-function-code \
		--function-name $(LAMBDA_FUNCTION_NAME) \
		--s3-bucket $(S3_LAMBDA_BUCKET) \
		--s3-key $(LAMBDA_PACKAGE) || \
	( \
		echo "Function doesn't exist, creating..." && \
		AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) AWS_DEFAULT_REGION=$(AWS_REGION) \
		aws --endpoint-url=$(LOCALSTACK_ENDPOINT) \
		lambda create-function \
		--function-name $(LAMBDA_FUNCTION_NAME) \
		--runtime provided.al2 \
		--role $(LAMBDA_ROLE_ARN) \
		--handler bootstrap \
		--code S3Bucket=$(S3_LAMBDA_BUCKET),S3Key=$(LAMBDA_PACKAGE) \
		--timeout $(LAMBDA_TIMEOUT) \
		--memory-size $(LAMBDA_MEMORY_SIZE) \
		--environment Variables='{$$(ENV_VARS)}' \
	)
	@echo "Lambda deployment complete!"

.PHONY: lambda-env
lambda-env:
	@echo "Updating Lambda environment variables..."
	@ENV_VARS=$$(make -s generate-env-string); \
	AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) AWS_DEFAULT_REGION=$(AWS_REGION) \
	aws --endpoint-url=$(LOCALSTACK_ENDPOINT) \
		lambda update-function-configuration \
		--function-name $(LAMBDA_FUNCTION_NAME) \
		--environment "Variables=$$ENV_VARS"

.PHONY: lambda-invoke
lambda-invoke:
	@echo "Invoking Lambda function..."
	@AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) AWS_DEFAULT_REGION=$(AWS_REGION) \
	aws --endpoint-url=$(LOCALSTACK_ENDPOINT) \
		lambda invoke \
		--function-name $(LAMBDA_FUNCTION_NAME) \
		--payload '{"test": true}' \
		response.json
	@cat response.json
	@rm response.json

.PHONY: lambda-logs
lambda-logs:
	@echo "Fetching Lambda logs..."
	@AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) AWS_DEFAULT_REGION=$(AWS_REGION) \
	aws --endpoint-url=$(LOCALSTACK_ENDPOINT) \
		logs tail /aws/lambda/$(LAMBDA_FUNCTION_NAME) --follow

# Generate environment variables string for Lambda from .env files
.PHONY: generate-env-string
generate-env-string:
	@{ \
		printf "{"; \
		first=true; \
		for file in .env.lambda .env.lambda.local; do \
			if [ -f "$$file" ]; then \
				while IFS='=' read -r key value; do \
					key=$$(echo "$$key" | sed 's/^[[:space:]]*//;s/[[:space:]]*$$//'); \
					if [ -n "$$key" ] && [ "$${key:0:1}" != "#" ]; then \
						value=$$(echo "$$value" | sed 's/^[[:space:]]*//;s/[[:space:]]*$$//;s/"//g'); \
						if [ "$$first" = false ]; then printf ","; fi; \
						printf "%s=%s" "$$key" "$$value"; \
						first=false; \
					fi; \
				done < "$$file"; \
			fi; \
		done; \
		printf "}"; \
	}

# Docker commands
docker-build:
	cd ../.. && docker build -f workers/downloader/Dockerfile -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: docker-build
	docker run --rm \
		--env-file $(PWD)/.env.local \
		--network host \
		-p 8080:8080 \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

# Code quality
fmt:
	go fmt ./...

lint:
	golangci-lint run

deps:
	go mod download
	go mod tidy