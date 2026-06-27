#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="${RUNWEAVER_BIN_DIR:-"$HOME/.local/bin"}"
BIN_PATH="$BIN_DIR/runweaver"

if ! command -v go >/dev/null 2>&1; then
  echo "runweaver install: Go is required to build runweaver" >&2
  exit 1
fi

GO_VERSION="$(go env GOVERSION 2>/dev/null || go version | awk '{print $3}')"
case "$GO_VERSION" in
  go1.25.[8-9]|go1.25.[1-9][0-9]|go1.2[6-9].*|go1.[3-9][0-9].*) ;;
  *)
    echo "warning: release builds should use Go 1.25.8 or newer; current toolchain is $GO_VERSION" >&2
    ;;
esac

mkdir -p "$BIN_DIR"
(
  cd "$ROOT_DIR"
  go build -o "$BIN_PATH" ./cmd/runweaver
)

echo "installed $BIN_PATH"
case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *)
    echo "warning: $BIN_DIR is not on PATH; add it before running runweaver or an AI coding runtime" >&2
    ;;
esac
