#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT}/pkg/a2a/proto"
OUT_DIR="${ROOT}/pkg/a2a/types"
GOOGLEAPIS_DIR="${A2A_GOOGLEAPIS_DIR:-${ROOT}/third_party/googleapis}"
TOOLS_DIR="${ROOT}/tools"
BIN_DIR="${TOOLS_DIR}/bin"
INCLUDE_DIR="${TOOLS_DIR}/include"

PROTOC_VERSION="${A2A_PROTOC_VERSION:-27.2}"
PROTOC_GEN_GO_VERSION="${A2A_PROTOC_GEN_GO_VERSION:-v1.36.10}"
PROTOC_GEN_GO_GRPC_VERSION="${A2A_PROTOC_GEN_GO_GRPC_VERSION:-v1.5.1}"

mkdir -p "${BIN_DIR}"
export PATH="${BIN_DIR}:${PATH}"

bootstrap_protoc() {
  if command -v protoc >/dev/null 2>&1 && [[ -d "${INCLUDE_DIR}/google/protobuf" ]]; then
    return 0
  fi

  if [[ "${A2A_BOOTSTRAP:-1}" == "0" ]]; then
    echo "protoc not found and A2A_BOOTSTRAP=0. Install protoc and retry." >&2
    exit 1
  fi

  if ! command -v curl >/dev/null 2>&1; then
    echo "curl not found. Install curl to bootstrap protoc." >&2
    exit 1
  fi
  if ! command -v unzip >/dev/null 2>&1; then
    echo "unzip not found. Install unzip to bootstrap protoc." >&2
    exit 1
  fi

  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "${arch}" in
    x86_64) arch="x86_64" ;;
    arm64) arch="aarch_64" ;;
    *) echo "unsupported arch: ${arch}" >&2; exit 1 ;;
  esac

  case "${os}" in
    darwin) os="osx" ;;
    linux) os="linux" ;;
    *) echo "unsupported os: ${os}" >&2; exit 1 ;;
  esac

  archive="protoc-${PROTOC_VERSION}-${os}-${arch}.zip"
  url="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${archive}"
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "${tmp_dir}"' EXIT

  echo "Downloading protoc ${PROTOC_VERSION}..."
  curl -fsSL "${url}" -o "${tmp_dir}/${archive}"
  unzip -q "${tmp_dir}/${archive}" -d "${tmp_dir}/protoc"
  cp -f "${tmp_dir}/protoc/bin/protoc" "${BIN_DIR}/protoc"
  chmod +x "${BIN_DIR}/protoc"
  mkdir -p "${INCLUDE_DIR}"
  cp -Rf "${tmp_dir}/protoc/include/." "${INCLUDE_DIR}/"
}

bootstrap_plugins() {
  if ! command -v protoc-gen-go >/dev/null 2>&1; then
    if [[ "${A2A_BOOTSTRAP:-1}" == "0" ]]; then
      echo "protoc-gen-go not found and A2A_BOOTSTRAP=0." >&2
      exit 1
    fi
    echo "Installing protoc-gen-go ${PROTOC_GEN_GO_VERSION}..."
    GOBIN="${BIN_DIR}" go install "google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}"
  fi

  if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
    if [[ "${A2A_BOOTSTRAP:-1}" == "0" ]]; then
      echo "protoc-gen-go-grpc not found and A2A_BOOTSTRAP=0." >&2
      exit 1
    fi
    echo "Installing protoc-gen-go-grpc ${PROTOC_GEN_GO_GRPC_VERSION}..."
    GOBIN="${BIN_DIR}" go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}"
  fi
}

bootstrap_protoc
bootstrap_plugins

mkdir -p "${OUT_DIR}"

if [[ ! -f "${GOOGLEAPIS_DIR}/google/api/annotations.proto" ]]; then
  echo "googleapis protos not found (expected ${GOOGLEAPIS_DIR}/google/api/annotations.proto)." >&2
  echo "Set A2A_GOOGLEAPIS_DIR to the googleapis proto root or vendor it under third_party/googleapis." >&2
  exit 1
fi

protoc \
  -I "${PROTO_DIR}" \
  -I "${GOOGLEAPIS_DIR}" \
  -I "${INCLUDE_DIR}" \
  --go_out="${OUT_DIR}" \
  --go_opt=paths=source_relative,Mpkg/a2a/proto/a2a.proto=github.com/jllopis/kairos/pkg/a2a/types \
  --go-grpc_out="${OUT_DIR}" \
  --go-grpc_opt=paths=source_relative,Mpkg/a2a/proto/a2a.proto=github.com/jllopis/kairos/pkg/a2a/types \
  "${PROTO_DIR}/a2a.proto"
