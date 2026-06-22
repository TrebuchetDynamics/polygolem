# Upstream Drift Runbook

Polygolem depends on Polymarket-owned Gamma, CLOB, Data API, and Relayer behavior. Fixtures catch local regressions; live drift still needs an operator loop.

## Weekly read-only smoke

Run from the repo root:

```bash
scripts/live-smoke.sh | tee smoke-$(date -u +%Y%m%dT%H%M%SZ).log
```

This default path is read-only: version, diagnostics, health, CLOB `/version`, and enriched crypto discovery. It should not require private keys or submit anything.

## Optional tiny live smoke

Only for a funded test wallet:

```bash
export POLYGOLEM_SMOKE_LIVE_ORDER=1
export POLYMARKET_PRIVATE_KEY=0x...
export POLYGOLEM_SMOKE_TOKEN_ID=<token>
export POLYGOLEM_SMOKE_PRICE=0.01
export POLYGOLEM_SMOKE_SIZE=1
export POLYGOLEM_MAX_LIVE_ORDER_USD=1
scripts/live-smoke.sh
```

The script refuses non-tiny smoke sizes/prices and stops on readiness failures.

## Opt-in live E2E

The live BTC 5m CLOB contract test is skipped by `go test -short ./...`. To run it manually:

```bash
go test ./tests -run TestBTCFiveMinuteLiveCLOBContracts -count=1 -v
```

Use it when CLOB/Gamma token mapping, book shape, fee fields, or market outcome behavior looks suspicious.

## When drift is suspected

1. Save the JSON envelope and exact command.
2. Compare the failing command with checked-in fixtures under `fixtures/`.
3. Run `scripts/live-smoke.sh` to identify which upstream surface changed.
4. If the change is real, update the smallest fixture/test first, then code.
5. Do not bypass deposit-wallet, readiness, or order-cap gates to work around upstream errors.
