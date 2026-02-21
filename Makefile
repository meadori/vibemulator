GO_VERSION ?= 1.25.5
GO_BINARY = vibemulator
GO_SOURCES = $(wildcard *.go) $(wildcard **/*.go)
GO_PACKAGES = ./...

.PHONY: all build run test clean deps check_go_version fmt rl-setup rl-train

all: build fmt

fmt:
	@echo "Formatting code..."
	@go fmt $(GO_PACKAGES)


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
	@go run nestest/main.go

rl-setup:
	@echo "Setting up Python Reinforcement Learning environment..."
	python3 -m venv venv
	. venv/bin/activate && pip install -r rl/requirements.txt
	. venv/bin/activate && python -m grpc_tools.protoc -I. --python_out=./rl --grpc_python_out=./rl api/controller.proto

rl-train: build
	@echo "Starting RL Training..."
	@echo "Make sure to run the emulator first in another terminal: ./vibemulator /path/to/game.nes"
	. venv/bin/activate && cd rl && python train_dqn.py

clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(GO_BINARY)
	@go clean
	@rm -rf nestest/testdata

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
