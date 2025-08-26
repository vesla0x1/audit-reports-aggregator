.PHONY: all build test clean run docker-build docker-run

# Variables
BINARY_NAME=downloader
DOCKER_IMAGE=audit-downloader
DOCKER_TAG=latest

# Build the binary
build:
	go build -o bin/$(BINARY_NAME) cmd/main.go

# Run tests
test:
	go test -v -cover ./...

# Run tests with coverage report
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run the application
run:
	go run cmd/main.go

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Run Docker container
docker-run:
	docker run --rm \
		--env-file .env \
		-p 8080:8080 \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy