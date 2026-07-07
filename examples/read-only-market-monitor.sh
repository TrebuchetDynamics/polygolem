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

pg --json health
pg --json discover crypto-window --asset "$asset" --interval "$interval" --enrich
