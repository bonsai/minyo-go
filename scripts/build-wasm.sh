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

OUT_DIR="$ROOT/web/dist"
mkdir -p "$OUT_DIR"
GOOS=js GOARCH=wasm go build -trimpath -o "$OUT_DIR/minyo-go.wasm" ./cmd/wasm
cp "$WASM_EXEC" "$OUT_DIR/wasm_exec.js"
cp "$ROOT/web/wasm-loader.js" "$OUT_DIR/wasm-loader.js"
cp "$ROOT/web/wasm-demo.html" "$OUT_DIR/index.html"
echo "WASM bundle created at web/dist/"
