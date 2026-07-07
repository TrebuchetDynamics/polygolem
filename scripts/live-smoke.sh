#!/usr/bin/env bash
set -euo pipefail

# Read-only by default. Set POLYGOLEM_SMOKE_LIVE_ORDER=1 plus the live-order
# variables below to place a tiny capped order.

pg() {
  if [[ -n "${POLYGOLEM_BIN:-}" ]]; then
    "$POLYGOLEM_BIN" "$@"
  else
    go run ./cmd/polygolem "$@"
  fi
}

run_json() {
  echo "\n==> polygolem $*" >&2
  pg --json "$@"
}

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "missing required env: $name" >&2
    exit 2
  fi
}

echo "polygolem live smoke: read-only phase" >&2
run_json version
run_json diag
run_json health
if command -v curl >/dev/null 2>&1; then
  clob_url="${POLYMARKET_CLOB_URL:-https://clob.polymarket.com}"
  echo "\n==> upstream CLOB version: ${clob_url}/version" >&2
  curl -fsS "${clob_url%/}/version" || echo '{"version":"unavailable"}'
  echo
fi
run_json discover crypto-5m --enrich

if [[ "${POLYGOLEM_SMOKE_LIVE_ORDER:-0}" != "1" ]]; then
  cat >&2 <<'MSG'

Read-only smoke complete. No private key, signing, order, relayer, or chain
mutation was required by this script.

To opt into one tiny live order, set all of:
  POLYGOLEM_SMOKE_LIVE_ORDER=1
  SIGNER_PRIVATE_KEY=0x...
  POLYGOLEM_SMOKE_TOKEN_ID=<clob token id>
  POLYGOLEM_SMOKE_PRICE=0.01
  POLYGOLEM_SMOKE_SIZE=1

The live phase runs readiness checks, submits a post-only limit order, prints the
result, and does not retry in another mode if anything fails.
MSG
  exit 0
fi

require_env SIGNER_PRIVATE_KEY
require_env POLYGOLEM_SMOKE_TOKEN_ID
: "${POLYGOLEM_SMOKE_PRICE:=0.01}"
: "${POLYGOLEM_SMOKE_SIZE:=1}"

case "$POLYGOLEM_SMOKE_PRICE" in
  0.0*|0.01|0.02|0.03|0.04|0.05) ;;
  *) echo "refusing smoke price above 0.05; set a tiny capped price" >&2; exit 2 ;;
esac
case "$POLYGOLEM_SMOKE_SIZE" in
  1|1.0|1.00) ;;
  *) echo "refusing smoke size other than 1" >&2; exit 2 ;;
esac

echo "\npolygolem live smoke: readiness + tiny post-only order" >&2
run_json auth status --check-deposit-key
run_json deposit-wallet status --check-enable-trading
run_json clob update-balance --asset-type collateral
run_json clob create-order \
  --token "$POLYGOLEM_SMOKE_TOKEN_ID" \
  --side buy \
  --price "$POLYGOLEM_SMOKE_PRICE" \
  --size "$POLYGOLEM_SMOKE_SIZE" \
  --post-only
