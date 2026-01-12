#!/usr/bin/env bash
set -euo pipefail

ADDR="${ADDR:-localhost:9030}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-60}"
QUERY="${1:-Cual fue el total de ventas en Q4 por region?}"

cd "$(dirname "${BASH_SOURCE[0]}")/.."

go run ./cmd/client --addr "$ADDR" --q "$QUERY" --timeout "$TIMEOUT_SECONDS"
