#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

QDRANT_URL="${QDRANT_URL:-localhost:6334}"
OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"
EMBED_MODEL="${EMBED_MODEL:-nomic-embed-text}"
KAIROS_LLM_MODEL="${KAIROS_LLM_MODEL:-qwen2.5-coder:7b-instruct-q5_K_M}"
KAIROS_LLM_PROVIDER="${KAIROS_LLM_PROVIDER:-ollama}"
CONFIG_PATH="${CONFIG_PATH:-}"

KNOWLEDGE_ADDR="${KNOWLEDGE_ADDR:-:9031}"
SPREADSHEET_ADDR="${SPREADSHEET_ADDR:-:9032}"
ORCH_ADDR="${ORCH_ADDR:-:9030}"
KNOWLEDGE_MCP_ADDR="${KNOWLEDGE_MCP_ADDR:-127.0.0.1:9041}"
SPREADSHEET_MCP_ADDR="${SPREADSHEET_MCP_ADDR:-127.0.0.1:9042}"

log() {
  printf "[%s] %s\n" "$(date +%H:%M:%S)" "$*"
}

check_port() {
  local name=$1
  local addr=$2
  local host=${addr%%:*}
  local port=${addr##*:}
  if ! (echo >"/dev/tcp/${host}/${port}") >/dev/null 2>&1; then
    log "WARN: ${name} not reachable at ${addr}"
    return 1
  fi
  return 0
}

check_http() {
  local name=$1
  local url=$2
  local hostport=${url#http://}
  hostport=${hostport#https://}
  local host=${hostport%%:*}
  local port=${hostport##*:}
  if ! (echo >"/dev/tcp/${host}/${port}") >/dev/null 2>&1; then
    log "WARN: ${name} not reachable at ${url}"
    return 1
  fi
  return 0
}

check_port "Qdrant gRPC" "$QDRANT_URL" || log "Hint: Qdrant gRPC defaults to localhost:6334."
check_http "Ollama HTTP" "$OLLAMA_URL" || log "Hint: Ollama defaults to http://localhost:11434."

log "Starting knowledge agent..."
config_args=()
if [[ -n "$CONFIG_PATH" ]]; then
  config_args=(--config "$CONFIG_PATH")
fi
( cd "$ROOT_DIR" && \
  OLLAMA_URL="$OLLAMA_URL" KAIROS_LLM_PROVIDER="$KAIROS_LLM_PROVIDER" KAIROS_LLM_MODEL="$KAIROS_LLM_MODEL" \
  go run ./cmd/knowledge --addr "$KNOWLEDGE_ADDR" --qdrant "$QDRANT_URL" --embed-model "$EMBED_MODEL" \
  "${config_args[@]}" --mcp-addr "$KNOWLEDGE_MCP_ADDR" ) &
KNOWLEDGE_PID=$!

log "Starting spreadsheet agent..."
( cd "$ROOT_DIR" && \
  OLLAMA_URL="$OLLAMA_URL" KAIROS_LLM_PROVIDER="$KAIROS_LLM_PROVIDER" KAIROS_LLM_MODEL="$KAIROS_LLM_MODEL" \
  go run ./cmd/spreadsheet --addr "$SPREADSHEET_ADDR" --data "$ROOT_DIR/data" --qdrant "$QDRANT_URL" \
  --embed-model "$EMBED_MODEL" "${config_args[@]}" --mcp-addr "$SPREADSHEET_MCP_ADDR" ) &
SPREADSHEET_PID=$!

log "Starting orchestrator..."
( cd "$ROOT_DIR" && \
  OLLAMA_URL="$OLLAMA_URL" KAIROS_LLM_PROVIDER="$KAIROS_LLM_PROVIDER" KAIROS_LLM_MODEL="$KAIROS_LLM_MODEL" \
  go run ./cmd/orchestrator --addr "$ORCH_ADDR" --knowledge "localhost${KNOWLEDGE_ADDR}" --spreadsheet "localhost${SPREADSHEET_ADDR}" \
  --qdrant "$QDRANT_URL" --embed-model "$EMBED_MODEL" "${config_args[@]}" ) &
ORCH_PID=$!

log "PIDs: knowledge=$KNOWLEDGE_PID spreadsheet=$SPREADSHEET_PID orchestrator=$ORCH_PID"
log "Press Ctrl+C to stop all."

trap 'log "Stopping..."; kill "$KNOWLEDGE_PID" "$SPREADSHEET_PID" "$ORCH_PID" 2>/dev/null || true' INT TERM
wait
