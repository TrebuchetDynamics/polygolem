# Polymarket Official APIs

## What this is

A map of Polymarket's official API architecture — Gamma, CLOB, Data API, plus the Relayer, Bridge, and WebSocket surfaces polygolem also uses — with links into the official documentation at [docs.polymarket.com](https://docs.polymarket.com) and citations to where each service is wired into this codebase.

## Start here

- New to Polymarket's API split? Read the [architecture overview](#api-architecture-overview) table below.
- Looking for which polygolem command hits which service? See [POLYMARKET-COVERAGE-MATRIX.md](./POLYMARKET-COVERAGE-MATRIX.md).
- Upstream behavior changed? Follow [UPSTREAM-DRIFT-RUNBOOK.md](./UPSTREAM-DRIFT-RUNBOOK.md).

## Discovering the official docs

The full official page index is machine-readable:

```
https://docs.polymarket.com/llms.txt
```

Every page also has a Markdown variant — append `.md` to any docs URL (e.g. `https://docs.polymarket.com/trading/overview.md`). Use this when re-verifying claims in this file.

## API architecture overview

Polymarket splits its platform across purpose-specific services. polygolem pins each default base URL centrally (`internal/cli/root.go:17-21`, `internal/config/config.go:38-39`, `pkg/universal/client.go:39-42`).

| Service | Base URL | Auth for reads | Purpose | Official entry point |
|---|---|---|---|---|
| Gamma API | `https://gamma-api.polymarket.com` | None | Market/event discovery: indexes on-chain + off-chain data into enriched market metadata, events, series, tags, comments, search | [Market data overview](https://docs.polymarket.com/market-data/overview) |
| CLOB API | `https://clob.polymarket.com` | None for market data; L1/L2 auth for account + trading | Central limit order book: books, prices, spreads, tick sizes, order placement/cancellation, trades | [Trading overview](https://docs.polymarket.com/trading/overview) |
| Data API | `https://data-api.polymarket.com` | None | User-centric data: positions, activity, holders, portfolio value, leaderboards, open interest | [Get positions](https://docs.polymarket.com/api-reference/core/get-current-positions-for-a-user) |
| Relayer V2 | `https://relayer-v2.polymarket.com` | SIWE-derived relayer credentials | Gasless deposit-wallet transactions: deploy, approvals, CTF redeem | [Relayer reference](https://docs.polymarket.com/api-reference/relayer/submit-a-transaction) |
| Bridge | `https://bridge.polymarket.com` | None for quotes/assets | Cross-chain deposits/withdrawals into Polymarket USD | [Bridge deposit](https://docs.polymarket.com/trading/bridge/deposit) |
| WebSocket (CLOB) | `wss://ws-subscriptions-clob.polymarket.com/ws/{market,user}` | None for market channel; L2 auth for user channel | Real-time book/price/trade events and authenticated order/trade updates | [WebSocket overview](https://docs.polymarket.com/market-data/websocket/overview) |

## Gamma API — market discovery

Read-only REST service for finding and describing markets. No authentication.

- polygolem client: `internal/gamma/client.go:15`, public SDK `pkg/gamma`, CLI `discover *` commands.
- Key official pages: [Fetching markets](https://docs.polymarket.com/market-data/fetching-markets), [List markets](https://docs.polymarket.com/api-reference/markets/list-markets), [List events](https://docs.polymarket.com/api-reference/events/list-events), [Tags](https://docs.polymarket.com/api-reference/tags/list-tags), [Search](https://docs.polymarket.com/api-reference/search/search-markets-events-and-profiles).
- Concepts: [Markets & Events](https://docs.polymarket.com/concepts/markets-events) explains the event → market → outcome-token hierarchy; a market's `conditionId` and two `clobTokenIds` are the join keys into the CLOB API.

## CLOB API — pricing and trading

The order book and matching engine. Market data endpoints are public; account and order endpoints require authentication.

- polygolem client: `internal/clob/client.go:21`, public SDK `pkg/clob/client.go:19`, CLI `clob *` and `orderbook *` commands.
- Two auth levels ([Authentication](https://docs.polymarket.com/api-reference/authentication)):
  - **L1** — EIP-712 signature from the wallet's private key; used to create/derive API keys and sign orders. polygolem's deposit-wallet POLY_1271 signing flow is documented in [POLY_1271-SIGNING.md](./POLY_1271-SIGNING.md).
  - **L2** — API key HMAC headers (`POLY_ADDRESS`, `POLY_SIGNATURE`, `POLY_TIMESTAMP`, `POLY_API_KEY`, `POLY_PASSPHRASE`); used for order posting/cancel and account reads.
- Key official pages: [Get order book](https://docs.polymarket.com/api-reference/market-data/get-order-book), [Prices history](https://docs.polymarket.com/api-reference/markets/get-prices-history), [Post a new order](https://docs.polymarket.com/api-reference/trade/post-a-new-order), [Cancel orders](https://docs.polymarket.com/api-reference/trade/cancel-multiple-orders), [Order lifecycle](https://docs.polymarket.com/concepts/order-lifecycle), [Rate limits](https://docs.polymarket.com/api-reference/rate-limits), [Error codes](https://docs.polymarket.com/resources/error-codes).
- Trading semantics: prices are 0–1 USD per share ([Prices & Orderbook](https://docs.polymarket.com/concepts/prices-orderbook)); fees per [Fees](https://docs.polymarket.com/trading/fees); geographic restrictions per [Geoblock](https://docs.polymarket.com/api-reference/geoblock).

## Data API — user positions and activity

Read-only REST service for wallet-level data. No authentication; keyed by wallet address.

- polygolem client: `internal/dataapi/doc.go:4`, public SDK `pkg/data/client.go:12`, CLI `data *` commands.
- Key official pages: [Current positions](https://docs.polymarket.com/api-reference/core/get-current-positions-for-a-user), [Closed positions](https://docs.polymarket.com/api-reference/core/get-closed-positions-for-a-user), [User activity](https://docs.polymarket.com/api-reference/core/get-user-activity), [Trades](https://docs.polymarket.com/api-reference/core/get-trades-for-a-user-or-markets), [Top holders](https://docs.polymarket.com/api-reference/core/get-top-holders-for-markets), [Leaderboard](https://docs.polymarket.com/api-reference/core/get-trader-leaderboard-rankings), [Open interest](https://docs.polymarket.com/api-reference/misc/get-open-interest).

## Relayer, Bridge, and on-chain surfaces

- **Relayer V2** submits gasless meta-transactions for deposit wallets (`internal/cli/deposit_wallet.go:56`, `pkg/relayer`). Official reference: [Submit a transaction](https://docs.polymarket.com/api-reference/relayer/submit-a-transaction), [Deposit Wallets](https://docs.polymarket.com/trading/deposit-wallets), [Gasless transactions](https://docs.polymarket.com/trading/gasless). polygolem's onboarding flow: [ONBOARDING.md](./ONBOARDING.md).
- **Bridge** handles deposits/withdrawals across chains into pUSD (`pkg/bridge/bridge.go:30`). Official: [Deposit](https://docs.polymarket.com/trading/bridge/deposit), [Supported assets](https://docs.polymarket.com/trading/bridge/supported-assets), [Polymarket USD](https://docs.polymarket.com/concepts/pusd).
- **Conditional Token Framework (CTF)** — outcome tokens are ERC-1155s split/merged/redeemed against USDC collateral ([CTF overview](https://docs.polymarket.com/trading/ctf/overview), [Redeem](https://docs.polymarket.com/trading/ctf/redeem)); polygolem helpers live in `pkg/ctf` and `pkg/settlement`. Contract addresses: [CONTRACTS.md](./CONTRACTS.md) and official [Contracts](https://docs.polymarket.com/resources/contracts).
- **Resolution** — markets resolve via UMA's optimistic oracle ([Resolution](https://docs.polymarket.com/concepts/resolution)); [Negative-risk markets](https://docs.polymarket.com/advanced/neg-risk) convert multi-outcome events.

## WebSocket — real-time data

- Market channel (public) and user channel (L2-authenticated) share the CLOB subscriptions host (`pkg/stream/client.go:16`, `pkg/stream/user.go:10`); CLI `stream market`, `stream user`, `marketdata live`.
- Official pages: [WebSocket overview](https://docs.polymarket.com/market-data/websocket/overview), [Market channel](https://docs.polymarket.com/market-data/websocket/market-channel), [User channel](https://docs.polymarket.com/market-data/websocket/user-channel), [Real-Time Data Socket](https://docs.polymarket.com/market-data/websocket/rtds).

## Additional official resources

- [Official GitHub](https://github.com/Polymarket) — open-source clients (`py-clob-client`, `clob-client` TypeScript) and examples; see also [Clients & SDKs](https://docs.polymarket.com/api-reference/clients-sdks), [Python SDK](https://docs.polymarket.com/dev-tooling/python), [TypeScript SDK](https://docs.polymarket.com/dev-tooling/typescript). polygolem tracks upstream deltas in [UPSTREAM-DRIFT-RUNBOOK.md](./UPSTREAM-DRIFT-RUNBOOK.md).
- [Help Center / Polymarket Learn](https://docs.polymarket.com/polymarket-learn) — beginner tutorials on trading and market mechanics (non-developer; `learn.polymarket.com` redirects here).
- [Polymarket 101](https://docs.polymarket.com/polymarket-101) and [Quickstart](https://docs.polymarket.com/quickstart) — official developer on-ramps.

## Update triggers

Revisit this page when:

- a default base URL changes in `internal/cli/root.go`, `internal/config/config.go`, or `pkg/universal/client.go`;
- `https://docs.polymarket.com/llms.txt` gains or removes a section relevant to a surface polygolem exposes;
- the live smoke script (`scripts/live-smoke.sh`) or drift runbook flags an upstream endpoint change;
- a new Polymarket service (host) is added to the codebase.

---

*Last verified against `docs.polymarket.com/llms.txt`: 2026-07-05*
