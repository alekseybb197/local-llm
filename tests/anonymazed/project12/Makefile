.PHONY: build test clean run deps

BUILD_DIR=bin
TEST_COVERAGE=coverage.out

build:
	@echo "Building OAuth2 Proxy..."
	go build -o $(BUILD_DIR)/proxy-server ./cmd/proxy-server
	@echo "Build complete: $(BUILD_DIR)/proxy-server"

test:
	@echo "Running tests..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=$(TEST_COVERAGE) -coverpkg=./... ./...
	@echo "Coverage report:"
	go tool cover -html=$(TEST_COVERAGE) -o coverage.html
	@echo "Open coverage.html in your browser"

clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)
	rm -f proxy.db
	rm -f $(TEST_COVERAGE) coverage.html
	@echo "Clean complete"

run: build
	@echo "Running OAuth2 Proxy..."
	./$(BUILD_DIR)/proxy-server

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies downloaded"
