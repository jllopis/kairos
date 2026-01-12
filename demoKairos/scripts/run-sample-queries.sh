#!/usr/bin/env bash
set -euo pipefail

ADDR="${ADDR:-localhost:9030}"
OUT_DIR="${OUT_DIR:-./outputs}"

mkdir -p "$OUT_DIR"

run_query() {
  local name=$1
  local query=$2
  local out="$OUT_DIR/${name}.txt"
  echo "==> ${query}" | tee "$out"
  go run ./cmd/client --addr "$ADDR" --q "$query" | tee -a "$out"
}

cd "$(dirname "${BASH_SOURCE[0]}")/.."

run_query "ventas_q4_region" "Cual fue el total de ventas en Q4 por region?"
run_query "top_margen" "Dame el top 10 de productos por margen y comparalo con el trimestre anterior"
run_query "anomalias_gastos" "Que anomalias hay en la hoja Gastos este mes?"

printf "\nResultados guardados en %s\n" "$OUT_DIR"
