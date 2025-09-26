# Binary names
SERVER_BINARY=server
CLIENT_BINARY=client

# Go command
GO=go

# Build directory
BUILD_DIR=build

# Source directories
SERVER_SRC=server/main.go
CLIENT_SRC=cmd/main.go

# Default target
.DEFAULT_GOAL := all

# Create build directory
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Build both server and client
.PHONY: all
all: $(BUILD_DIR) build-server build-client

# Build server
.PHONY: build-server
build-server: $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(SERVER_BINARY) $(SERVER_SRC)

# Build client
.PHONY: build-client
build-client: $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(CLIENT_BINARY) $(CLIENT_SRC)

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# Run tests
.PHONY: test
test:
	$(GO) test ./...