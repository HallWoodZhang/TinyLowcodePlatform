#!/bin/sh
# Apply QuickJS darwin/amd64 CGO fix to GOPATH module cache.
# Fixes: https://github.com/quickjs-go/quickjs-go (unreported)
# CGO directive for darwin/amd64 points to arm64 lib, correct path is x86_64/

set -e

QUICKJS_VER="v0.0.0-20230414054158-b72900cb68c1"
CACHE_DIR="${GOMODCACHE:-$(go env GOMODCACHE)}/github.com/quickjs-go/quickjs-go@${QUICKJS_VER}"

if [ ! -f "$CACHE_DIR/quickjs.go" ]; then
    echo "quickjs-go not found in module cache, downloading..."
    GOPROXY="${GOPROXY:-https://goproxy.cn,direct}" go mod download github.com/quickjs-go/quickjs-go@${QUICKJS_VER}
fi

TARGET="$CACHE_DIR/quickjs.go"
WRONG="#cgo darwin,amd64 LDFLAGS: -L\${SRCDIR}/3rdparty/libs/quickjs/darwin -lquickjs"
RIGHT="#cgo darwin,amd64 LDFLAGS: -L\${SRCDIR}/3rdparty/libs/quickjs/darwin/x86_64 -lquickjs"

if grep -qF "$RIGHT" "$TARGET" 2>/dev/null; then
    echo "quickjs-go darwin/amd64 patch already applied"
    exit 0
fi

chmod -R u+w "$CACHE_DIR" 2>/dev/null || true
sed -i '' -e "s|$WRONG|$RIGHT|" "$TARGET" 2>/dev/null || \
    sed -i    -e "s|$WRONG|$RIGHT|" "$TARGET"

go clean -cache 2>/dev/null || true
echo "quickjs-go darwin/amd64 patch applied to $TARGET"
