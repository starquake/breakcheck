SHELL=/bin/bash -o pipefail

.PHONY: all
all: clean lint build test

.PHONY: clean
clean:
	rm -rf bin
	mkdir -p bin

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
