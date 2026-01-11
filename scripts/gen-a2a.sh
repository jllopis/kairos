#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT}/docs/protocols/A2A"
OUT_DIR="${ROOT}/pkg/a2a/types"
GOOGLEAPIS_DIR="${A2A_GOOGLEAPIS_DIR:-${ROOT}/third_party/googleapis}"

if ! command -v protoc >/dev/null 2>&1; then
  echo "protoc not found. Install protoc and retry." >&2
  exit 1
fi

if ! command -v protoc-gen-go >/dev/null 2>&1; then
  echo "protoc-gen-go not found. Install google.golang.org/protobuf/cmd/protoc-gen-go." >&2
  exit 1
fi

if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
  echo "protoc-gen-go-grpc not found. Install google.golang.org/grpc/cmd/protoc-gen-go-grpc." >&2
  exit 1
fi

mkdir -p "${OUT_DIR}"

if [[ ! -f "${GOOGLEAPIS_DIR}/google/api/annotations.proto" ]]; then
  echo "googleapis protos not found (expected ${GOOGLEAPIS_DIR}/google/api/annotations.proto)." >&2
  echo "Set A2A_GOOGLEAPIS_DIR to the googleapis proto root." >&2
  exit 1
fi

protoc \
  -I "${PROTO_DIR}" \
  -I "${GOOGLEAPIS_DIR}" \
  --go_out="${OUT_DIR}" \
  --go_opt=paths=source_relative,Mdocs/protocols/A2A/a2a.proto=github.com/jllopis/kairos/pkg/a2a/types \
  --go-grpc_out="${OUT_DIR}" \
  --go-grpc_opt=paths=source_relative,Mdocs/protocols/A2A/a2a.proto=github.com/jllopis/kairos/pkg/a2a/types \
  "${PROTO_DIR}/a2a.proto"
