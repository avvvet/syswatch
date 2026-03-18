BINARY_SERVER=syswatch-server
BUILD_DIR=bin
CMD_SERVER=./cmd/server

# Go build flags — strip debug info for smaller binary
LDFLAGS=-ldflags="-s -w"

.PHONY: all build clean tidy fmt vet run help

## all: tidy, vet and build
all: tidy vet build

## build: build the server binary
build:
	@echo "Building $(BINARY_SERVER)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_SERVER) $(CMD_SERVER)
	@echo "Done: $(BUILD_DIR)/$(BINARY_SERVER)"
	@ls -lh $(BUILD_DIR)/$(BINARY_SERVER)

## run: run directly with go run
run:
	@go run $(CMD_SERVER)

## tidy: tidy go modules
tidy:
	@echo "Tidying modules..."
	@go mod tidy

## fmt: format all Go files
fmt:
	@echo "Formatting..."
	@gofmt -w .

## vet: run go vet
vet:
	@echo "Vetting..."
	@go vet ./...

## clean: remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

## help: show this help
help:
	@echo "Usage:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'