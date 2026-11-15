#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

GOROOT="$(go env GOROOT)"
if [[ -f "$GOROOT/lib/wasm/wasm_exec.js" ]]; then
  WASM_EXEC="$GOROOT/lib/wasm/wasm_exec.js"
elif [[ -f "$GOROOT/misc/wasm/wasm_exec.js" ]]; then
  WASM_EXEC="$GOROOT/misc/wasm/wasm_exec.js"
else
  echo "error: wasm_exec.js not found under GOROOT ($GOROOT)" >&2
  exit 1
fi

mkdir -p web/public
cp "$WASM_EXEC" web/public/wasm_exec.js
GOOS=js GOARCH=wasm go build -o web/public/ahorita.wasm ./cmd/wasm
ls -lh web/public/ahorita.wasm web/public/wasm_exec.js
