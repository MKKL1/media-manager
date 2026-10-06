#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../plugins/wasm"
for dir in */; do
  name=${dir%/}
  (cd "$name" && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o "$name.wasm" .)
  echo "built plugins/wasm/$name/$name.wasm"
done
