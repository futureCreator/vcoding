BINARY_NAME=vcoding
MAIN_PATH=./cmd/vcoding
DIST_DIR=dist

GO=$(shell which go)
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w"

.PHONY: all build build-all clean test help

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME) $(MAIN_PATH)

build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	@echo "  -> linux/amd64"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@echo "  -> linux/arm64"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	@echo "  -> darwin/amd64"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	@echo "  -> darwin/arm64"
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "Done!"

clean:
	@echo "Cleaning..."
	rm -rf $(DIST_DIR)

test:
	$(GO) test -v ./...

help:
	@echo "Usage:"
	@echo "  make build      - Build for current platform"
	@echo "  make build-all  - Build for all platforms (linux/darwin, amd64/arm64)"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make test       - Run tests"
