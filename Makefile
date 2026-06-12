.PHONY: all build run clean fmt vet lint test

BINARY_DIR := cmd/ts-quickjs/bin
BINARY     := $(BINARY_DIR)/ts-quickjs
CMD_DIR    := ./cmd/ts-quickjs

all: build

build:
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY) $(CMD_DIR)

run: build
	./$(BINARY)

clean:
	rm -rf $(BINARY_DIR) build

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test ./...
