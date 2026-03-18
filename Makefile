BINARY   := notion
MODULE   := github.com/paoloanzn/neo-notion-cli
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE     := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GOBIN    ?= $(shell go env GOPATH)/bin

LDFLAGS  := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: build install uninstall clean test lint fmt vet

## build: Compile the binary into ./bin/
build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .
	@echo "Built bin/$(BINARY)"

## install: Build and install the binary to GOBIN ($(GOBIN))
install:
	go build -ldflags "$(LDFLAGS)" -o $(GOBIN)/$(BINARY) .
	@echo "Installed $(GOBIN)/$(BINARY)"

## uninstall: Remove the binary from GOBIN
uninstall:
	rm -f $(GOBIN)/$(BINARY)
	@echo "Removed $(GOBIN)/$(BINARY)"

## clean: Remove build artifacts
clean:
	rm -rf bin/
	go clean

## test: Run all tests
test:
	go test -race -count=1 ./...

## lint: Run go vet and staticcheck (if available)
lint: vet
	@command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

## vet: Run go vet
vet:
	go vet ./...

## fmt: Format all Go source files
fmt:
	gofmt -s -w .

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
