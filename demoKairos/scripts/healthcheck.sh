#!/usr/bin/env bash
set -euo pipefail

KNOWLEDGE_ADDR="${KNOWLEDGE_ADDR:-localhost:9031}"
SPREADSHEET_ADDR="${SPREADSHEET_ADDR:-localhost:9032}"
ORCH_ADDR="${ORCH_ADDR:-localhost:9030}"

check_port() {
  local name=$1
  local addr=$2
  local host=${addr%%:*}
  local port=${addr##*:}
  if (echo >"/dev/tcp/${host}/${port}") >/dev/null 2>&1; then
    echo "[ok] ${name} ${addr}"
  else
    echo "[fail] ${name} ${addr}" >&2
    return 1
  fi
}

check_port "knowledge" "$KNOWLEDGE_ADDR"
check_port "spreadsheet" "$SPREADSHEET_ADDR"
check_port "orchestrator" "$ORCH_ADDR"
