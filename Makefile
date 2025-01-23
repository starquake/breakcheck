SHELL=/bin/bash -o pipefail

.PHONY: all
all: clean lint build test

.PHONY: clean
clean:
	rm -rf bin
	mkdir -p bin

.PHONY: lint
lint:
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix

.PHONY: build
build:
	mkdir -p bin
	go build -v -o bin/ ./...

.PHONY: test
test:
	go test -v ./...
