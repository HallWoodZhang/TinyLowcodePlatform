#!/bin/bash
# quick_dev_start.sh — Start all 5 Tiny Lowcode Platform services locally
# Usage: ./quick_dev_start.sh [start|stop|status]

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
export GOPROXY

JWT_SECRET_FILE="$SCRIPT_DIR/.jwt_secret"

# ─── helpers ──────────────────────────────────────────────

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; NC='\033[0m'
ok()  { echo -e "${GREEN}✅${NC} $1"; }
info(){ echo -e "${CYAN}ℹ${NC}  $1"; }
err() { echo -e "${RED}❌${NC} $1"; }

check_bin() {
    command -v "$1" >/dev/null 2>&1 || { err "$1 not found, please install first"; exit 1; }
}

kill_port() {
    local port=$1
    local pid=$(lsof -ti tcp:$port 2>/dev/null || true)
    if [ -n "$pid" ]; then
        info "Killing process on port $port (pid $pid)"
        kill "$pid" 2>/dev/null || true
        sleep 0.5
    fi
}

wait_for() {
    local port=$1 url=$2 label=$3 max=15
    info "Waiting for $label on :$port ..."
    for i in $(seq 1 $max); do
        local code=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
        if [ "$code" != "000" ]; then
            ok "$label ready (:${port})"
            return 0
        fi
        sleep 0.5
    done
    err "$label failed to start on :$port"
    return 1
}

# ─── stop ─────────────────────────────────────────────────

do_stop() {
    info "Stopping all services..."
    for port in 9720 9721 9722 9723 9724; do
        kill_port $port
    done
    ok "All services stopped"
}

# ─── status ───────────────────────────────────────────────

do_status() {
    echo ""
    for svc in "9720 ts-quickjs" "9721 sql-runner" "9722 auth-server" "9723 admin-server" "9724 bff-server"; do
        set -- $svc
        port=$1; name=$2
        if lsof -ti tcp:$port >/dev/null 2>&1; then
            printf "  ${GREEN}●${NC} %-15s :%s\n" "$name" "$port"
        else
            printf "  ${RED}○${NC} %-15s :%s  (stopped)\n" "$name" "$port"
        fi
    done
    echo ""
}

# ─── start ────────────────────────────────────────────────

do_start() {
    echo ""
    info "Tiny Lowcode Platform v2.0.0 Gateway"
    echo ""

    # 1. checks
    check_bin go

    # 2. clean up old processes
    for port in 9720 9721 9722 9723 9724; do
        kill_port $port
    done

    # 3. generate shared JWT secret
    if [ ! -f "$JWT_SECRET_FILE" ]; then
        info "Generating shared JWT secret..."
        openssl rand -base64 32 > "$JWT_SECRET_FILE"
    fi
    JWT_SECRET=$(cat "$JWT_SECRET_FILE" | tr -d '\n')
    # encode as JSON-safe string (escapes newlines etc)
    JWT_SECRET_JSON=$(echo "$JWT_SECRET" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read()))" 2>/dev/null || echo "\"$JWT_SECRET\"")

    # 4. create config files from samples
    for svc in auth-server admin-server bff-server ts-quickjs sql-runner; do
        conf="cmd/$svc/conf/config.json"
        sample="cmd/$svc/conf/config.json.sample"
        if [ ! -f "$conf" ]; then
            info "Creating $conf from sample..."
            cp "$sample" "$conf"
        fi
        # inject JWT secret into all configs
        python3 -c "
import json, sys
with open('$conf') as f:
    cfg = json.load(f)
cfg['jwt_secret'] = $JWT_SECRET_JSON
with open('$conf', 'w') as f:
    json.dump(cfg, f, indent=2)
" 2>/dev/null || true
    done

    # 5. apply QuickJS patch
    info "Applying QuickJS patch..."
    bash scripts/patch-quickjs.sh 2>/dev/null || true

    # 6. build
    info "Building all services..."
    mkdir -p cmd/auth-server/bin cmd/admin-server/bin cmd/bff-server/bin cmd/ts-quickjs/bin cmd/sql-runner/bin
    go build -o cmd/auth-server/bin/auth-server   ./cmd/auth-server/   && ok "auth-server"   || err "auth-server build failed"
    go build -o cmd/admin-server/bin/admin-server ./cmd/admin-server/  && ok "admin-server"  || err "admin-server build failed"
    go build -o cmd/bff-server/bin/bff-server     ./cmd/bff-server/    && ok "bff-server"    || err "bff-server build failed"
    go build -o cmd/sql-runner/bin/sql-runner     ./cmd/sql-runner/    && ok "sql-runner"    || err "sql-runner build failed"
    go build -o cmd/ts-quickjs/bin/ts-quickjs     ./cmd/ts-quickjs/    && ok "ts-quickjs"    || err "ts-quickjs build failed"

    # 7. start services
    echo ""
    info "Starting services..."
    LOWCODE_HOME="$SCRIPT_DIR" cmd/auth-server/bin/auth-server   > /tmp/auth-server.log   2>&1 &
    LOWCODE_HOME="$SCRIPT_DIR" cmd/admin-server/bin/admin-server > /tmp/admin-server.log  2>&1 &
    LOWCODE_HOME="$SCRIPT_DIR" cmd/bff-server/bin/bff-server     > /tmp/bff-server.log    2>&1 &
    LOWCODE_HOME="$SCRIPT_DIR" cmd/ts-quickjs/bin/ts-quickjs     > /tmp/ts-quickjs.log    2>&1 &
    LOWCODE_HOME="$SCRIPT_DIR" cmd/sql-runner/bin/sql-runner     > /tmp/sql-runner.log    2>&1 &

    sleep 1

    # 8. health check
    wait_for 9722 "http://127.0.0.1:9722/api/auth/login"   "auth-server"
    wait_for 9724 "http://127.0.0.1:9724/api/bff/version"   "bff-server"
    wait_for 9720 "http://127.0.0.1:9720/api/scripts"       "ts-quickjs"
    wait_for 9721 "http://127.0.0.1:9721/api/sql/tables"     "sql-runner"
    wait_for 9723 "http://127.0.0.1:9723/api/admin/tenants"  "admin-server"

    # 9. start Vue frontend dev server
    if [ -d "$SCRIPT_DIR/frontend/node_modules" ]; then
        info "Starting Vue frontend dev server..."
        cd "$SCRIPT_DIR/frontend" && npm run dev -- --host 0.0.0.0 > /tmp/vite-dev.log 2>&1 &
        cd "$SCRIPT_DIR"
        sleep 2
        wait_for 5173 "http://127.0.0.1:5173" "Vue frontend"
    fi

    # 10. summary
    echo ""
    echo "  ┌─────────────────────────────────────────────┐"
    echo "  │  Tiny Lowcode Platform v2.0.0 Gateway       │"
    echo "  ├─────────────────────────────────────────────┤"
    echo "  │  Frontend: http://127.0.0.1:5173            │"
    echo "  │  ───────────────────────────────────────    │"
    echo "  │  Auth API:  http://127.0.0.1:9722           │"
    echo "  │  BFF API:   http://127.0.0.1:9724           │"
    echo "  │  Admin API: http://127.0.0.1:9723           │"
    echo "  │  Scripts:   http://127.0.0.1:9720           │"
    echo "  │  SQL:       http://127.0.0.1:9721           │"
    echo "  ├─────────────────────────────────────────────┤"
    echo "  │  Default:  admin / admin / admin123         │"
    echo "  │  Logs:     /tmp/*-server.log                │"
    echo "  │  Stop:     ./quick_dev_start.sh stop        │"
    echo "  └─────────────────────────────────────────────┘"
    echo ""
}

# ─── main ─────────────────────────────────────────────────

case "${1:-start}" in
    start)  do_start  ;;
    stop)   do_stop   ;;
    status) do_status ;;
    restart) do_stop; do_start ;;
    *)
        echo "Usage: $0 {start|stop|status|restart}"
        exit 1
        ;;
esac
