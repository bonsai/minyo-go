#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"

GOROOT=$(go env GOROOT)
WASM_EXEC="$GOROOT/lib/wasm/wasm_exec.js"
if [ ! -f "$WASM_EXEC" ]; then
  WASM_EXEC="$GOROOT/misc/wasm/wasm_exec.js"
fi
if [ ! -f "$WASM_EXEC" ]; then
  echo "wasm_exec.js not found in Go toolchain: $GOROOT" >&2
  exit 1
fi

# The Go API serves ./wasm; static hosting uses the same assets under public/.
mkdir -p "$ROOT/wasm" "$ROOT/public/wasm"
GOOS=js GOARCH=wasm go build -trimpath -o "$ROOT/wasm/minyo-go.wasm" ./cmd/wasm
cp "$WASM_EXEC" "$ROOT/wasm/wasm_exec.js"
cp "$ROOT/web/wasm-loader.js" "$ROOT/wasm/wasm-loader.js"
cp "$ROOT/wasm/"* "$ROOT/public/wasm/"
echo "WASM assets created in wasm/ and public/wasm/"
