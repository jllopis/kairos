#!/usr/bin/env bash
set -euo pipefail

ADDR="${ADDR:-localhost:9030}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-60}"
OUT_DIR="${OUT_DIR:-./outputs/$(date -u +%Y%m%dT%H%M%SZ)}"

mkdir -p "$OUT_DIR"

run_query() {
  local name=$1
  local query=$2
  local out="$OUT_DIR/${name}.txt"
  local started
  started=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  echo "==> ${query}" | tee "$out"
  go run ./cmd/client --addr "$ADDR" --q "$query" --timeout "$TIMEOUT_SECONDS" | tee -a "$out"
  local finished
  finished=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  echo "- ${name} | ${started} -> ${finished} | ${out}" >> "$OUT_DIR/summary.md"
}

cd "$(dirname "${BASH_SOURCE[0]}")/.."

printf "# demoKairos sample run\n\n" > "$OUT_DIR/summary.md"
printf "Generated: %s\n\n" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" >> "$OUT_DIR/summary.md"

run_query "ventas_q4_region" "Cual fue el total de ventas en Q4 por region?"
run_query "top_margen" "Dame el top 10 de productos por margen y comparalo con el trimestre anterior"
run_query "anomalias_gastos" "Que anomalias hay en la hoja Gastos este mes?"

printf "\nResultados guardados en %s\n" "$OUT_DIR"
