GO_VERSION ?= 1.25.5
GO_BINARY = vibemulator
GO_SOURCES = $(wildcard *.go) $(wildcard **/*.go)
GO_PACKAGES = ./...

.PHONY: all build run test clean deps check_go_version

all: build

build: deps
	@echo "Building $(GO_BINARY)..."
	@go build -o $(GO_BINARY) .

run: build
	@echo "Running $(GO_BINARY)..."
	@./$(GO_BINARY) $(ROM_FILE)

test: deps
	@echo "Running tests..."
	@go test $(GO_PACKAGES)

nestest: deps
	@echo "Running nestest CPU test..."
	@go run nestest/main.go ~/Stuff/roms/nestest.nes

clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(GO_BINARY)
	@go clean

deps:
	@echo "Ensuring Go modules are downloaded..."
	@go mod download
	@go mod tidy

# Helper to ensure the correct Go version is used, if not already
check_go_version:
	@echo "Checking Go version..."
	@go_current_version=$$(go version | awk '{print $$3}' | sed 's/go//'); \
	if [ "$$go_current_version" != "$(GO_VERSION)" ]; then \
		echo "WARNING: Your Go version ($$go_current_version) does not match the recommended version ($(GO_VERSION))."; \
		echo "         This might cause build issues. Consider installing Go $(GO_VERSION)."; \
	fi
