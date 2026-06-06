.PHONY: build test clean run docker-build docker-run docker-up docker-down

# Build variables
BINARY=main
GO=go
GOFLAGS=-mod=readonly

# Build the application
build:
	@echo "Building $(BINARY)..."
	CGO_ENABLED=1 go build -o $(BINARY) .
	@echo "Build complete: ./$(BINARY)"

# Run tests
test: test-unit

# Unit tests
test-unit:
	@echo "Running unit tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "Tests completed. Coverage report: coverage.out"

# Generate coverage report
coverage: test-unit
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run the application
run: build
	@echo "Running $(BINARY) on port 8080..."
	./$(BINARY) -config ./config.json || ./$(BINARY)

# Development mode with hot reload support
dev: build
	@echo "Running in development mode..."
	@echo "To stop: press Ctrl+C"
	./$(BINARY) -config ./config.json

# Docker build
docker-build:
	@echo "Building Docker image..."
	 docker build -t oauth2-llm-proxy:latest .
	@echo "Docker image built: oauth2-llm-proxy:latest"

# Docker run
docker-run: docker-build
	@echo "Running Docker container..."
	 docker run -d --name oauth2-llm-proxy \
		-p 8080:8080 \
		-e LLM_URL=http://localhost:11434 \
		oauth2-llm-proxy:latest
	@echo "Container running: oauth2-llm-proxy"

# Docker up with docker-compose
docker-up:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d
	@echo "Services started"

# Docker down
docker-down:
	@echo "Stopping services..."
	docker-compose down
	@echo "Services stopped"

# Docker clean
docker-clean:
	@echo "Cleaning Docker containers and volumes..."
	docker-compose down -v
	docker rmi oauth2-llm-proxy:latest || true
	@echo "Docker cleaned"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY)
	rm -f coverage.out coverage.html
	rm -rf data/*
	@echo "Clean complete"

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies installed"

# Lint
lint:
	@echo "Running linter..."
	@golangci-lint run || echo "golangci-lint not installed, skipping"

# Format
format:
	@echo "Formatting code..."
	go fmt ./...

# Help
help:
	@echo "OAuth2 Proxy for LLM - Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  build              - Build the application"
	@echo "  test               - Run all tests"
	@echo "  test-unit          - Run unit tests with coverage"
	@echo "  coverage           - Generate coverage report"
	@echo "  run                - Run the application"
	@echo "  dev                - Development mode"
	@echo "  deps               - Install dependencies"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-run         - Run Docker container"
	@echo "  docker-up          - Start with docker-compose"
	@echo "  docker-down        - Stop with docker-compose"
	@echo "  docker-clean       - Clean Docker containers and volumes"
	@echo "  clean              - Clean build artifacts"
	@echo "  lint               - Run linter"
	@echo "  format             - Format code"
	@echo "  help               - Show this help message"
