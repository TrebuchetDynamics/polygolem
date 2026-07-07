# Polygolem 5-Minute Crypto Markets Guide

This guide is for end users who want to try Polygolem on Polymarket's fast crypto up/down markets. You can discover live markets and run a paper-trade demo without a wallet, private key, or funded account.

> **Not financial advice.** The commands below are for market discovery, tooling checks, and paper trading. Do not use live trading commands until you understand the deposit-wallet flow and your own risk limits.

## What you will do

1. Install or build the `polygolem` CLI.
2. Check that public Polymarket APIs are reachable.
3. List the current 5-minute crypto markets.
4. Inspect one BTC 5-minute window.
5. Simulate an UP or DOWN paper trade.
6. Review your local paper positions.

Supported 5-minute assets: BTC, ETH, SOL, XRP, BNB, DOGE, and HYPE.

## 1. Install

Use the released binary path if you are an end user:

```bash
go install github.com/TrebuchetDynamics/polygolem/cmd/polygolem@latest
```

Or build from a checkout:

```bash
git clone https://github.com/TrebuchetDynamics/polygolem
cd polygolem
go build -o polygolem ./cmd/polygolem
```

Verify the CLI can reach public APIs:

```bash
polygolem health --json
```

No credentials are required for the rest of this demo.

## 2. List every active 5-minute crypto market

```bash
# Current window only
polygolem discover crypto-5m --json

# Current + next hour, with local display fields
polygolem discover crypto-5m --hours-ahead 1 --timezone America/Denver --json

# Optional: focus on the liquid majors
polygolem discover crypto-5m --asset BTC --asset ETH --asset SOL --hours-ahead 1 --enrich --json
```

What to look for in the JSON response:

- `data.interval`: should be `5m`.
- `data.window_start`: the UTC start of the current 5-minute window.
- `data.markets[].asset`: one of BTC, ETH, SOL, XRP, BNB, DOGE, HYPE.
- `data.markets[].status`: `active` means Polygolem found an open accepting market.
- `data.markets[].window_start_local`: local time when `--timezone` is set.
- `data.markets[].token_ids`: CLOB token IDs for the UP/DOWN outcomes.
- `data.markets[].liquidity_clob`, `best_bid`, `best_ask`, and `book_spread`: Gamma book/liquidity snapshot fields.
- `data.markets[].price` and `spread`: live CLOB fields included when `--enrich` succeeds.

If a market returns `not_found` or `no_active_market`, wait for the next 5-minute UTC boundary and run the command again. These markets are short-lived and can appear slightly after the boundary.

For a simple 5m-only strategy scan, start with BTC/ETH/SOL rows where `book_spread <= 0.01`, `liquidity_clob >= 2000`, and the target window is current or next. Use limit orders; market buys can cross thin books.

## 3. Inspect one market window

Use `crypto-window` when you know the asset and interval:

```bash
polygolem discover crypto-window --asset BTC --interval 5m --enrich --json
```

This bypasses fuzzy search. Polygolem computes the deterministic Polymarket slug for the current UTC window, then fetches the exact event.

Useful fields:

- `data.slug`: the resolved window slug.
- `data.markets[0].question`: the market question shown on Polymarket.
- `data.markets[0].outcomes`: usually UP/DOWN-style outcomes.
- `data.markets[0].token_ids`: token IDs used for order books, prices, and paper trades.

## 4. Paper trade the current BTC 5-minute window

Reset the local paper account to make the demo easy to read:

```bash
polygolem paper reset --cash 100 --json
```

Simulate buying one UP share in the current BTC 5-minute window:

```bash
polygolem paper trade --asset BTC --interval 5m --side up --size 1 --json
```

Or simulate the DOWN side:

```bash
polygolem paper trade --asset BTC --interval 5m --side down --size 1 --json
```

Review your local paper state:

```bash
polygolem paper positions --json
```

Paper trades are local simulations backed by live market data. They do not submit orders, move funds, or require `POLYMARKET_PRIVATE_KEY`.

## 5. Optional: inspect raw order-book data

If you want to inspect one token directly, copy a token ID from `data.markets[0].token_ids` and run:

```bash
polygolem orderbook get --token-id <TOKEN_ID> --json
```

Use this when you want to compare the paper-trade price with the current book.

## 6. Moving from demo to live trading

Live trading is intentionally separate from this demo. Before placing a real order, read:

- [README: Trade in Four Commands](README.md#trade-in-four-commands)
- [Live Trade Walkthrough](docs/LIVE-TRADE-WALKTHROUGH.md)
- [Safety Model](docs/SAFETY.md)

Live trading requires `POLYMARKET_PRIVATE_KEY`, a V2 deposit wallet, and explicit order commands. Market discovery and paper trading stay read-only by default.

## Troubleshooting

| Symptom | What to do |
|---|---|
| `window not found` | Wait for the next 5-minute UTC boundary and retry. The exact slug may not exist yet. |
| `no_active_market` | The event exists, but the market is closed or inactive. Retry the next window. |
| No `price` or `spread` | Re-run with `--enrich`; if still absent, CLOB enrichment may be temporarily unavailable. |
| Paper trade uses a fallback-looking price | Inspect the token with `orderbook get`; live price lookup may have failed. |
| You see an authentication error | Ensure you are using only the read-only commands in this guide; none should require credentials. |

## One-screen demo

```bash
polygolem health --json
polygolem discover crypto-5m --enrich --json
polygolem discover crypto-window --asset BTC --interval 5m --enrich --json
polygolem paper reset --cash 100 --json
polygolem paper trade --asset BTC --interval 5m --side up --size 1 --json
polygolem paper positions --json
```
