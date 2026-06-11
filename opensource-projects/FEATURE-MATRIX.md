# Polymarket Open-Source Repo Feature Matrix

This matrix summarizes the repositories under `opensource-projects/repos/`: what each project does, how it overlaps with the others, and where it is unique.

## Sources reviewed

- Repo READMEs and manifests (`README.md`, `go.mod`, `package.json`, `pyproject.toml`, `Cargo.toml`, `foundry.toml`).
- Source layout and examples under each repo (`client/`, `pkg/`, `py_clob_client/`, `src/`, `examples/`, smart-contract `src/`).
- Nested vendored submodule READMEs under `ctf-exchange-v2/lib/` and `polymarket-kit/vendors/`.
- Archive/deprecation notices in `py-clob-client` and `rs-clob-client` READMEs.

## Legend

- `✅` implemented / first-class
- `◐` partial, narrow, example-only, or wrapper-level
- `—` not a focus
- `⚠️` archived/deprecated upstream notice

## High-level inventory

| Repo | Language / runtime | Primary purpose | Best fit | Maintenance signal from repo |
|---|---:|---|---|---|
| `ctf-exchange-v2` | Solidity / Foundry | Polymarket CTF Exchange V2 smart contracts, collateral wrapper, adapters, order matching. | Protocol truth source for V2 order semantics, settlement, collateral, addresses, gas snapshots. | Active-looking V2 protocol repo with audits listed. |
| `Polymarket-golang` | Go | V2-native Go CLOB SDK aligned with `py-clob-client-v2`; includes WS, bridge, rewards, gasless on-chain operations. | Rich Go reference for current V2 signing/trading/on-chain flows. | README says V2-native and current. |
| `foxme666-Polymarket-golang` | Go | Older Go SDK implementing core `py-clob-client` CLOB features. | Simpler Go port / legacy baseline. | Local tracked snapshot; README targets `py-clob-client` V1 style. |
| `polymarket-go-sdk` | Go | Layered, production-oriented Go SDK for CLOB, auth, transport, execution, WS, RTDS, Gamma/Data/Bridge/CTF. | Most architected Go foundation with tests, execution lifecycle, retries. | Community-maintained, spec-aligned; v2 module path. |
| `polymarket-go` | Go | Broad community SDK covering CLOB, Relayer, Data, Gamma, Bridge, WS, Turnkey. | Broad API coverage and Turnkey wallet integration. | Community / third-party disclaimer. |
| `go-builder-signing-sdk` | Go | Small official-ish builder authentication header SDK. | Reusable HMAC/builder-header implementation for Go clients. | Narrow Polymarket SDK. |
| `polymarket-go-gamma-client` | Go | Gamma REST API client for market/event metadata and discovery. | Read-only market discovery, analytics, opportunity scans. | Focused maintained-style client. |
| `polymarket_cli` | Go CLI | Small JSON CLI for Gamma search, market lookup, CLOB orderbook. | Agent/LLM tool shelling out to Polymarket data. | Lightweight utility. |
| `polymarket-kit` | TypeScript + Python + Go | Typed SDK plus Elysia/OpenAPI proxy, MCP server, TS/Python/Go clients, WS utilities. | Multi-language/proxy/MCP experiments and type-normalized API surface. | Work-in-progress per README. |
| `py-clob-client` | Python | Original Python CLOB client with auth, signing, order mgmt, RFQ examples/tests. | Legacy compatibility and golden behavior reference. | ⚠️ Archived; README says migrate to new unified SDK. |
| `rs-clob-client` | Rust | Typed Rust SDK for CLOB plus optional WS, RTDS, Data, Gamma, Bridge, RFQ, CTF. | Rust reference for type-state auth and feature-gated clients. | ⚠️ Archived; README says migrate to V2 Rust client. |

## Nested vendored submodules

These are real nested git repositories, but they are dependencies of top-level repos rather than independent Polymarket client candidates.

| Nested repo | Parent | What it does | Why it matters here |
|---|---|---|---|
| `ctf-exchange-v2/lib/forge-std` | `ctf-exchange-v2` | Foundry standard library for Solidity tests, scripts, and cheatcodes. | Test/deployment dependency, not a Polymarket API surface. |
| `ctf-exchange-v2/lib/solady` | `ctf-exchange-v2` | Gas-optimized Solidity contracts and libraries including EIP-712/ECDSA/ERC1271 primitives. | Protocol dependency for efficient smart-contract utilities/signature helpers. |
| `polymarket-kit/vendors/builder-signing-sdk` | `polymarket-kit` | TypeScript SDK for local/remote Polymarket builder header signing. | TS equivalent/companion to the Go `go-builder-signing-sdk`. |
| `polymarket-kit/vendors/clob-client-fork` | `polymarket-kit` | Fork of the TypeScript `@polymarket/clob-client` V1-style client. | Legacy TS CLOB implementation used by the kit/proxy experiments. |
| `polymarket-kit/vendors/clob-client-v2` | `polymarket-kit` | TypeScript `@polymarket/clob-client-v2` for current CLOB V2 order/auth flows. | Current TS CLOB reference; README recommends newer unified `Polymarket/ts-sdk` for new projects. |
| `polymarket-kit/vendors/real-time-data-client` | `polymarket-kit` | TypeScript websocket client for real-time data streams: activity, comments, crypto/equity prices, etc. | RTDS dependency/reference for non-CLOB real-time streams. |

## Feature matrix

| Capability | ctf-exchange-v2 | Polymarket-golang | foxme666-Polymarket-golang | polymarket-go-sdk | polymarket-go | go-builder-signing-sdk | polymarket-go-gamma-client | polymarket_cli | polymarket-kit | py-clob-client | rs-clob-client |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| CLOB REST market data | — | ✅ | ✅ | ✅ | ✅ | — | — | ◐ orderbook | ✅ | ✅ | ✅ |
| CLOB order placement/cancel/query | — | ✅ | ✅ | ✅ | ✅ | — | — | — | ◐ | ✅ | ✅ |
| V2 CTF Exchange order signing | Protocol | ✅ | — | ◐ | ◐ | — | — | — | ◐ Go/TS order utils | — | ◐ |
| EIP-712 signing | Contract verifies | ✅ | ✅ | ✅ | ✅ | — | — | — | ✅ | ✅ | ✅ |
| L2 HMAC/API-key auth | — | ✅ | ✅ | ✅ | ✅ | — | — | — | ✅ | ✅ | ✅ |
| Builder attribution/auth headers | Order fields | ✅ | — | ✅ | ✅ optional | ✅ | — | — | ✅ | ✅ examples | ✅ |
| RFQ | — | ✅ | ✅ | ✅ | — | — | — | — | — | ✅ | ✅ tests/examples |
| WebSocket market/user streams | — | ✅ | — | ✅ | ✅ market data | — | — | — | ✅ | — | ✅ |
| Gamma market/event metadata | — | ✅ | — | ✅ | ✅ | — | ✅ | ✅ search/market | ✅ | — | ✅ |
| Data API / positions/activity | — | ✅ | — | ✅ | ✅ | — | — | — | ✅ | — | ✅ |
| Bridge/deposits | — | ✅ | — | ✅ | ✅ | — | — | — | — | — | ✅ |
| CTF split/merge/redeem client ops | Smart contracts | ✅ gasless + direct | ✅ examples | ✅ | — | — | — | — | — | — | ✅ |
| Collateral wrap/onramp/offramp | Smart contracts | ✅ pUSD wrap | — | ◐ CTF/onchain | — | — | — | — | — | — | ◐ approvals/CTF |
| Rewards/rebates | — | ✅ | — | ◐ rewards allowance example | ◐ rewards/notifications | — | — | — | — | ✅ scoring/notifs | — |
| Relayer/Safe support | — | ◐ gasless web3 | ◐ gasless web3 | — | ✅ Relayer + Safe helpers | — | — | — | — | — | — |
| Turnkey wallet integration | — | — | — | — | ✅ | — | — | — | — | — | — |
| AWS KMS / remote signer | — | — | — | ✅ AWS KMS | — | ◐ remote builder signer | — | — | — | — | ✅ AWS example |
| Proxy server / OpenAPI | — | — | — | — | — | — | — | — | ✅ Elysia/OpenAPI | — | — |
| MCP server / AI tooling | — | — | — | — | — | — | — | ◐ Claude skill | ✅ MCP | — | — |
| CLI binary | — | — | — | ◐ bot/cmd tools | — | — | — | ✅ | ◐ bin scripts | — | — |
| Strong type/runtime validation | Solidity types | Go types | Go types | Go types + tests | Go types | Go types | Go types | Go structs | ✅ TypeBox/Zod | Python models | Rust types/type-state |

## What the repos share

### Shared domain model

Most client repos model the same Polymarket building blocks:

- **CLOB REST**: health/server time, markets, order books, prices/spreads, trades, order creation/post/cancel/query.
- **Authentication tiers**: read-only access, EIP-712 L1 signing with a private key/signer, L2 API credentials with HMAC request signing.
- **Order types and sides**: BUY/SELL, limit/market orders, GTC/GTD/FOK/FAK/post-only variants depending on repo age.
- **Proxy/funder concepts**: funder/proxy/deposit wallet address distinct from signer address appears in Python, Go ports, Rust, and V2 Go repos.
- **Builder attribution**: builder headers or builder fields show up in `go-builder-signing-sdk`, `Polymarket-golang`, `polymarket-go-sdk`, `polymarket-go`, `polymarket-kit`, `py-clob-client`, and `rs-clob-client`.
- **Polymarket endpoints**: `clob.polymarket.com`, `gamma-api.polymarket.com`, `data-api.polymarket.com`, `bridge.polymarket.com`, and websocket endpoints recur across SDKs.

### Shared implementation patterns

- **EIP-712 + HMAC signing helpers** are repeated in Python, Go, Rust, and TypeScript codebases.
- **Order builders** exist in `py-clob-client`, both `Polymarket-golang` variants, `polymarket-go-sdk`, `polymarket-kit` Go client, and `rs-clob-client`.
- **Typed request/response structs** are a main selling point for the Go/Rust/TypeScript clients.
- **Examples-first docs** are common: almost every SDK ships runnable examples for orderbooks, auth, order placement, or data discovery.
- **Tests mirror protocol risks**: signing, headers, order builder rounding, websocket reconnects, RFQ payloads, and transport behavior are frequently tested where repos are mature.

### Shared gaps/risks

- **Version drift**: V1 clients (`py-clob-client`, `foxme666-Polymarket-golang`, possibly parts of older broad SDKs) may not match V2 CTF Exchange order formats.
- **Archive notices**: upstream `py-clob-client` and `rs-clob-client` explicitly warn they are archived/deprecated.
- **Duplicate signing logic**: many repos reimplement EIP-712/HMAC/builder signing; this is useful for cross-checking but creates drift risk.
- **Secret-handling risk**: many examples require private keys/API keys; use environment variables and never commit credentials.
- **Partial API coverage varies**: focused clients (`go-builder-signing-sdk`, `polymarket-go-gamma-client`, `polymarket_cli`) intentionally omit trading or on-chain flows.

## Repo-by-repo notes

### `ctf-exchange-v2`

Protocol-level Solidity implementation. It defines the V2 exchange, collateral token, onramp/offramp, CTF adapters, and matching flow. It is the best source for exact V2 semantics: `matchOrders`, signature verification, PMCT/pUSD collateral, complementary/mint/merge settlement, preapproval, user pause, and gas snapshots. Client SDKs should be checked against this repo when V2 order signing or collateral behavior matters.

### `Polymarket-golang`

The richest current Go port in this folder. Its README says it is V2-native, aligned with `py-clob-client-v2`, and byte-for-byte tested against golden signatures. It covers CLOB L0/L1/L2, V2 order creation/posting, builder code/API-key attribution, RFQ, market/user websocket clients, bridge, Data/Gamma APIs, rewards/rebates, and gasless web3 operations such as split/merge/redeem/convert/wrap USDC.e to pUSD. It shares the older `py-clob-client` style API shape but adds current V2 modules.

### `foxme666-Polymarket-golang`

Older Go SDK and likely ancestor/snapshot of the V2 Go port. It implements core `py-clob-client`-style CLOB flows: auth levels, order management, orderbooks/prices/spreads, RFQ, EIP-712 signing, plus examples for split/merge/redeem/gasless operations. It appears useful as a simpler/legacy comparison, but not as the best current V2 reference.

### `polymarket-go-sdk`

A production-architecture Go SDK. It has layered packages for auth, CLOB, transport, execution, websocket, RTDS, Data, Gamma, Bridge, CTF, logging, errors, and bot/risk modules. Its distinctive value is engineering structure: retry policy, rate limiting, circuit breaker, error normalization, execution lifecycle state, idempotency, websocket policy, and broad tests. It overlaps heavily with `Polymarket-golang` but is more framework-like.

### `polymarket-go`

Broad third-party Go SDK. It covers CLOB, relayer, Data API, Gamma API, Bridge, websocket market data, signing, headers, HMAC/EIP-712 helpers, and Turnkey wallet management. Its unique feature in this set is Turnkey integration and Relayer/Safe helper focus. README warns it is unofficial and gas-less transactions require deriving a Gnosis Safe wallet before trading.

### `go-builder-signing-sdk`

Small focused Go library for Polymarket builder headers. It has local and remote signer examples and HMAC helpers. It does not try to be a CLOB client; instead it is a reusable auth component that larger SDKs can depend on or cross-check against.

### `polymarket-go-gamma-client`

Focused read-only Go client for Gamma REST metadata. It covers markets, events, series, search, sports, tags, and health. Its examples are analytics/opportunity scanners: wide spreads, low liquidity/high volume, new markets, related-market arbitrage, negative-risk opportunities, rapid price movement, and soon-closing markets. It shares Gamma types and filters with broader SDKs but avoids trading/auth complexity.

### `polymarket_cli`

Small Go CLI designed to emit clean JSON for scripts and AI agents. Commands cover Gamma search, market details, and CLOB orderbook. It overlaps with the read-only parts of SDKs, but its value is operational: easy shell integration and a packaged Claude skill.

### `polymarket-kit`

Multi-runtime toolkit: TypeScript SDK, Elysia proxy server with OpenAPI, MCP server, Python package, and Go client. It emphasizes type safety and normalization with TypeBox/Zod schemas, generated/parallel APIs, websocket clients, and a proxy layer over Gamma/CLOB/Data. It overlaps broadly with SDKs but is most unique as a typed proxy/MCP/multi-language experiment.

### `py-clob-client`

Original Python CLOB client with extensive examples and tests for CLOB auth, signing, headers, order creation/posting/canceling, balances/allowances, notifications/scoring, and RFQ. The README warns it is archived and no longer functional for new/existing integrations, but it remains useful as a legacy behavioral reference and for understanding the API shape many ports copied.

### `rs-clob-client`

Rust SDK with feature flags for `clob`, `ws`, `rtds`, `data`, `gamma`, `bridge`, `rfq`, `heartbeats`, and `ctf`. It has strong type-state authentication and async `reqwest` design, plus examples for AWS signing, builder auth, CLOB streaming, approvals, bridge, CTF, Data, Gamma, and RTDS. README warns it is archived and points to a V2 Rust client, so treat it as a legacy/typed-reference repo rather than current production source.

## Quick selection guide

| If you need... | Start with |
|---|---|
| Exact V2 contract/order/collateral semantics | `ctf-exchange-v2` |
| Current Go V2 trading + gasless CTF operations | `Polymarket-golang` |
| Go SDK architecture with retries/execution lifecycle/tests | `polymarket-go-sdk` |
| Turnkey / relayer / broad Go API surface | `polymarket-go` |
| Builder auth headers only | `go-builder-signing-sdk` |
| Gamma-only market discovery in Go | `polymarket-go-gamma-client` |
| JSON CLI for agents/scripts | `polymarket_cli` |
| TypeScript proxy/OpenAPI/MCP/multi-language toolkit | `polymarket-kit` |
| Legacy Python CLOB behavior/tests | `py-clob-client` |
| Legacy Rust typed design/reference | `rs-clob-client` |

## Cross-repo consolidation opportunities

1. **Signing golden tests**: centralize fixtures for EIP-712, HMAC, builder headers, V2 CTF signatures, and compare Go/Python/Rust/TS outputs.
2. **Endpoint/schema drift watch**: use `polymarket-kit` OpenAPI/type schemas or `polymarket-go-sdk` tests as a drift detector for all clients.
3. **Current vs legacy labeling**: clearly mark `py-clob-client`, `rs-clob-client`, and older `foxme666-Polymarket-golang` as legacy/V1-oriented in local docs.
4. **Reusable builder auth**: treat `go-builder-signing-sdk` as the Go source of truth for builder headers and compare with larger SDK implementations.
5. **Protocol alignment checks**: validate all V2 clients against `ctf-exchange-v2` structs and order lifecycle, especially builder/metadata fields, collateral token assumptions, and EIP-1271/deposit-wallet signing.
