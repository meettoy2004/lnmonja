.PHONY: all build clean server agent cli test fmt vet

VERSION ?= dev
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Detect OS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    # macOS: Disable CGO for gopsutil compatibility
    export CGO_ENABLED=0
endif

all: build

build: server agent cli

server:
	@echo "Building lnmonja-server..."
	@go build $(LDFLAGS) -o lnmonja-server ./cmd/lnmonja-server

agent:
	@echo "Building lnmonja-agent..."
	@CGO_ENABLED=0 go build $(LDFLAGS) -o lnmonja-agent ./cmd/lnmonja-agent

cli:
	@echo "Building lnmonja-cli..."
	@go build $(LDFLAGS) -o lnmonja-cli ./cmd/lnmonja-cli

clean:
	@echo "Cleaning build artifacts..."
	@rm -f lnmonja-server lnmonja-agent lnmonja-cli

test:
	@echo "Running tests..."
	@go test -v ./...

fmt:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Vetting code..."
	@go vet ./...

# Install binaries to GOPATH/bin
install: build
	@echo "Installing binaries..."
	@cp lnmonja-server $(GOPATH)/bin/
	@cp lnmonja-agent $(GOPATH)/bin/
	@cp lnmonja-cli $(GOPATH)/bin/

# Build for Linux (useful when building on macOS for deployment)
build-linux:
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o lnmonja-server-linux ./cmd/lnmonja-server
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o lnmonja-agent-linux ./cmd/lnmonja-agent
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o lnmonja-cli-linux ./cmd/lnmonja-cli

# Build for macOS
build-darwin:
	@echo "Building for macOS..."
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o lnmonja-server-darwin ./cmd/lnmonja-server
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o lnmonja-agent-darwin ./cmd/lnmonja-agent
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o lnmonja-cli-darwin ./cmd/lnmonja-cli

# Build for all platforms
build-all: build-linux build-darwin
	@echo "Built for all platforms"
