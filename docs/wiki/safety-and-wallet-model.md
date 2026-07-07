# Safety and Wallet Model

## What this is

A compact map of Polygolem's live-money safety posture. Use this before editing
`wallet`, `exchange`, signing, funding, or confirmation behavior.

## Start here

The live path is deposit-wallet-only: the `wallet` command says Polymarket V2
requires a deposit wallet as order maker/signer while the EOA remains the
signing key (`internal/cli/deposit_wallet.go:59` to
`internal/cli/deposit_wallet.go:64`). The lifecycle is derive → deploy → approve
→ fund → trade → redeem (`internal/cli/deposit_wallet.go:63` to
`internal/cli/deposit_wallet.go:64`).

## Live-money guard seam

`internal/livegate` is the shared guard package. Its package comment says it
centralizes the checks that must fire before signing or submitting a live-money
transaction: per-order notional cap and typed confirmation token
(`internal/livegate/livegate.go:1` to `internal/livegate/livegate.go:7`).

| Guard | Source evidence | Meaning |
|---|---|---|
| Default order cap | `DefaultMaxLiveOrderUSD = "1"` (`internal/livegate/livegate.go:21` to `internal/livegate/livegate.go:22`) | If `POLYGOLEM_MAX_LIVE_ORDER_USD` is unset, live orders are capped at 1 pUSD/USDC equivalent. |
| Cap enforcement | `EnforceNotionalCap` reads `POLYGOLEM_MAX_LIVE_ORDER_USD` and rejects notional above cap (`internal/livegate/livegate.go:24` to `internal/livegate/livegate.go:39`) | Order commands must call this before loading a private key. |
| Typed confirmation | `RequireConfirm` requires an exact token match (`internal/livegate/livegate.go:42` to `internal/livegate/livegate.go:49`) | Wallet mutations cannot submit from one mistyped flag. |

## Wallet command posture

The wallet help separates read-only commands (`derive`, `status`, `nonce`,
`redeemable`, `settlement-status`) from live-money commands (`deploy`,
`approve`, `approve-adapters`, `batch`, `fund`, `onboard`, `redeem`) and states
that live submissions require typed `--confirm` tokens
(`internal/cli/deposit_wallet.go:66` to `internal/cli/deposit_wallet.go:74`).

## Exchange command posture

The exchange help separates read-only CLOB data (`book`, `markets`, `market`,
`market-by-token`, `price-history`, `simulate`, `tick-size`) from authenticated
account reads and live order placement (`internal/cli/cmd_clob.go:254` to
`internal/cli/cmd_clob.go:267`). Live order placement enforces
`POLYGOLEM_MAX_LIVE_ORDER_USD` before signing (`internal/cli/cmd_clob.go:265` to
`internal/cli/cmd_clob.go:267`).

## Regression net

`TestLiveMoneyCommandsFailClosed` enumerates live-money commands and proves cap
or confirmation guards fire before private-key loading (`internal/cli/livegate_coverage_test.go:10`
to `internal/cli/livegate_coverage_test.go:27`). Current checked rows include
`exchange create-order`, `exchange market-order`, `exchange batch-orders`, and
wallet submission commands (`internal/cli/livegate_coverage_test.go:41` to
`internal/cli/livegate_coverage_test.go:51`).

## Update triggers

Refresh this page when:

- `internal/livegate/livegate.go` changes;
- a new mutating command is added under `wallet` or `exchange`;
- `internal/cli/livegate_coverage_test.go` changes;
- docs/SAFETY.md or docs/SAFE-HAPPY-PATH.md changes live-money guidance.
