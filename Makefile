SHELL=/bin/bash -o pipefail

.PHONY: all
all: lint build test

.PHONY: lint
lint:
	@echo "Starting formatting..."
	golangci-lint run
	@echo "Finished formatting..."

.PHONY: lint-fix
lint-fix:
	@echo "Starting formatting..."
	golangci-lint run --fix
	@echo "Finished formatting..."

.PHONY: build
build:
	mkdir -p bin
	go build -v -o bin/ ./...

.PHONY: test
test:
	go test -v ./...
