.PHONY: all build clean fmt vet lint test
.PHONY: ts sql build-ts build-sql run-ts run-sql clean-ts clean-sql

SVC ?=

TS_DIR      := cmd/ts-quickjs
TS_BIN_DIR  := $(TS_DIR)/bin
TS_BIN      := $(TS_BIN_DIR)/ts-quickjs
SQL_DIR     := cmd/sql-runner
SQL_BIN_DIR := $(SQL_DIR)/bin
SQL_BIN     := $(SQL_BIN_DIR)/sql-runner

all: build

# --- parameterized build: make SVC=ts-quickjs / make SVC=sql-runner ---
build:
ifeq ($(SVC),ts-quickjs)
	@mkdir -p $(TS_BIN_DIR)
	go build -o $(TS_BIN) ./$(TS_DIR)
else ifeq ($(SVC),sql-runner)
	@mkdir -p $(SQL_BIN_DIR)
	go build -o $(SQL_BIN) ./$(SQL_DIR)
else
	@mkdir -p $(TS_BIN_DIR)
	go build -o $(TS_BIN) ./$(TS_DIR)
	@mkdir -p $(SQL_BIN_DIR)
	go build -o $(SQL_BIN) ./$(SQL_DIR)
endif

# --- short aliases ---
ts build-ts:
	@mkdir -p $(TS_BIN_DIR)
	go build -o $(TS_BIN) ./$(TS_DIR)

sql build-sql:
	@mkdir -p $(SQL_BIN_DIR)
	go build -o $(SQL_BIN) ./$(SQL_DIR)

# --- run ---
run-ts: ts
	./$(TS_BIN)

run-sql: sql
	./$(SQL_BIN)

# --- clean ---
clean:
	rm -rf $(TS_BIN_DIR) $(SQL_BIN_DIR) build

clean-ts:
	rm -rf $(TS_BIN_DIR)

clean-sql:
	rm -rf $(SQL_BIN_DIR)

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test ./...

COVER_PKGS := ./core/config/... ./core/handler/...
COVER_OUT  := coverage.out
COVER_HTML := coverage.html

cover:
	go test -coverprofile=$(COVER_OUT) $(COVER_PKGS)
	go tool cover -func=$(COVER_OUT) | tail -1
	@echo ""
	@echo "Coverage report → $(COVER_HTML)"
	go tool cover -html=$(COVER_OUT) -o $(COVER_HTML)

cover-summary:
	go test -coverprofile=$(COVER_OUT) $(COVER_PKGS)
	@echo ""
	@echo "==================== Coverage Summary ===================="
	@go tool cover -func=$(COVER_OUT) | awk ' \
		/total:/ { printf "\n  TOTAL: %s\n\n", $$3 } \
		!/total:/ && !/^_/ { printf "  %-50s %s\n", $$1, $$3 }'
	@echo "=========================================================="
	@rm -f $(COVER_OUT)
