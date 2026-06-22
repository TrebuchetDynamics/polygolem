#!/usr/bin/env bash
set -euo pipefail

pg() {
  if [[ -n "${POLYGOLEM_BIN:-}" ]]; then
    "$POLYGOLEM_BIN" "$@"
  else
    go run ./cmd/polygolem "$@"
  fi
}

asset="${1:-BTC}"
interval="${2:-5m}"
side="${3:-up}"
size="${4:-1}"

pg --json paper reset --cash 100
pg --json paper trade --asset "$asset" --interval "$interval" --side "$side" --size "$size"
pg --json paper positions
