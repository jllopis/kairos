#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

QDRANT_URL="${QDRANT_URL:-http://localhost:6333}"
OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"
EMBED_MODEL="${EMBED_MODEL:-nomic-embed-text}"

KNOWLEDGE_ADDR="${KNOWLEDGE_ADDR:-:9031}"
SPREADSHEET_ADDR="${SPREADSHEET_ADDR:-:9032}"
ORCH_ADDR="${ORCH_ADDR:-:9030}"

log() {
  printf "[%s] %s\n" "$(date +%H:%M:%S)" "$*"
}

log "Starting knowledge agent..."
( cd "$ROOT_DIR" && go run ./cmd/knowledge --addr "$KNOWLEDGE_ADDR" --qdrant "$QDRANT_URL" --embed-model "$EMBED_MODEL" ) &
KNOWLEDGE_PID=$!

log "Starting spreadsheet agent..."
( cd "$ROOT_DIR" && go run ./cmd/spreadsheet --addr "$SPREADSHEET_ADDR" --data "$ROOT_DIR/data" ) &
SPREADSHEET_PID=$!

log "Starting orchestrator..."
( cd "$ROOT_DIR" && go run ./cmd/orchestrator --addr "$ORCH_ADDR" --knowledge "localhost${KNOWLEDGE_ADDR}" --spreadsheet "localhost${SPREADSHEET_ADDR}" --qdrant "$QDRANT_URL" --embed-model "$EMBED_MODEL" ) &
ORCH_PID=$!

log "PIDs: knowledge=$KNOWLEDGE_PID spreadsheet=$SPREADSHEET_PID orchestrator=$ORCH_PID"
log "Press Ctrl+C to stop all."

trap 'log "Stopping..."; kill "$KNOWLEDGE_PID" "$SPREADSHEET_PID" "$ORCH_PID" 2>/dev/null || true' INT TERM
wait
