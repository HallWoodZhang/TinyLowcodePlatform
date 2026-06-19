.PHONY: all build clean fmt vet lint test
.PHONY: auth admin bff ts sql run-auth run-admin run-bff run-ts run-sql

VERSION ?= v2.0.0
BUILD_TIME = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS = -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

AUTH_DIR   := cmd/auth-server
AUTH_BIN   := $(AUTH_DIR)/bin/auth-server
ADMIN_DIR  := cmd/admin-server
ADMIN_BIN  := $(ADMIN_DIR)/bin/admin-server
BFF_DIR    := cmd/bff-server
BFF_BIN    := $(BFF_DIR)/bin/bff-server
TS_DIR     := cmd/ts-quickjs
TS_BIN     := $(TS_DIR)/bin/ts-quickjs
SQL_DIR    := cmd/sql-runner
SQL_BIN    := $(SQL_DIR)/bin/sql-runner

all: build

build: auth admin bff ts sql

auth:
	@mkdir -p $(AUTH_DIR)/bin
	go build -ldflags "$(LDFLAGS)" -o $(AUTH_BIN) ./$(AUTH_DIR)

admin:
	@mkdir -p $(ADMIN_DIR)/bin
	go build -ldflags "$(LDFLAGS)" -o $(ADMIN_BIN) ./$(ADMIN_DIR)

bff:
	@mkdir -p $(BFF_DIR)/bin
	go build -ldflags "$(LDFLAGS)" -o $(BFF_BIN) ./$(BFF_DIR)

ts:
	@mkdir -p $(TS_DIR)/bin
	go build -ldflags "$(LDFLAGS)" -o $(TS_BIN) ./$(TS_DIR)

sql:
	@mkdir -p $(SQL_DIR)/bin
	go build -ldflags "$(LDFLAGS)" -o $(SQL_BIN) ./$(SQL_DIR)

# --- cross-platform builds (Linux amd64/arm64) ---
build-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o $(TS_BIN)-linux-amd64 ./$(TS_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(AUTH_BIN)-linux-amd64 ./$(AUTH_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(ADMIN_BIN)-linux-amd64 ./$(ADMIN_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BFF_BIN)-linux-amd64 ./$(BFF_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(SQL_BIN)-linux-amd64 ./$(SQL_DIR)

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o $(TS_BIN)-linux-arm64 ./$(TS_DIR)
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(AUTH_BIN)-linux-arm64 ./$(AUTH_DIR)
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(ADMIN_BIN)-linux-arm64 ./$(ADMIN_DIR)
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BFF_BIN)-linux-arm64 ./$(BFF_DIR)
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(SQL_BIN)-linux-arm64 ./$(SQL_DIR)

run-auth: auth
	./$(AUTH_BIN)

run-admin: admin
	./$(ADMIN_BIN)

run-bff: bff
	./$(BFF_BIN)

run-ts: ts
	./$(TS_BIN)

run-sql: sql
	./$(SQL_BIN)

clean:
	rm -rf $(AUTH_DIR)/bin $(ADMIN_DIR)/bin $(BFF_DIR)/bin $(TS_DIR)/bin $(SQL_DIR)/bin

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test ./core/... ./tests/... -v -count=1

cover:
	go test -coverprofile=coverage.out ./core/... ./tests/...
	go tool cover -html=coverage.out -o coverage.html

# --- docker ---
docker-build:
	docker build -t auth-server -f $(AUTH_DIR)/Dockerfile .
	docker build -t admin-server -f $(ADMIN_DIR)/Dockerfile .
	docker build -t bff-server -f $(BFF_DIR)/Dockerfile .
	docker build -t ts-quickjs -f $(TS_DIR)/Dockerfile .
	docker build -t sql-runner -f $(SQL_DIR)/Dockerfile .

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down
