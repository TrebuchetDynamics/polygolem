# Architecture

## What this is

`polygolem` is a Go SDK and CLI interface into Polymarket APIs and contracts.
The CLI is a thin shell over typed, testable internal packages and a public SDK
in `pkg/`.

## Start here

- Need CLI wiring? Start at `internal/cli/root.go:58`; command handlers are documented as thin delegates in `internal/cli/doc.go:1`.
- Need public import boundaries? Start with the `pkg/` table below and [ADR-0004](adr/0004-public-sdk-boundary.md).
- Need safety boundaries? Read [SAFETY.md](SAFETY.md), then the mode and signature sections below.

## Source anchors

| Claim | Source |
|---|---|
| The root Cobra command owns global `--json` and command registration. | `internal/cli/root.go:58`, `internal/cli/root.go:79`, `internal/cli/root.go:104` |
| CLI handlers should delegate; default mode is read-only; live uses deposit-wallet signing gates. | `internal/cli/doc.go:1`, `internal/cli/doc.go:4`, `internal/cli/doc.go:5` |
| Contract approval capability sets live in `pkg/contracts`. | `pkg/contracts/contracts.go:96`, `pkg/contracts/contracts.go:104`, `pkg/contracts/contracts.go:116`, `pkg/contracts/contracts.go:122` |
| MCP/OpenAPI are intentionally read-only agent surfaces. | `pkg/mcp/mcp.go:1`, `pkg/mcp/mcp.go:86`, `pkg/openapi/openapi.go:1` |

## Surface map

### Public SDK (`pkg/`)

Stable interfaces for downstream Go consumers (e.g., `go-bot`).

| Package | Purpose |
|---|---|
| `pkg/bridge` | Bridge API client — supported assets, deposit addresses, quotes. |
| `pkg/builder` | Builder attribution and fee configuration for order placement. |
| `pkg/capabilities` | Typed Capability Map: per-surface service, auth requirement, wallet mode, and read-only/mutating classification (`pkg/capabilities/capabilities.go:1`). |
| `pkg/clob` | CLOB market-data plus stable authenticated account/order transaction DTOs. |
| `pkg/compat` | Machine-readable compatibility contract assembled from the Capability Map and error kinds (`pkg/compat/compat.go:1`). |
| `pkg/contracts` | Polygon contract registry, wallet approval capability sets, and contract-code deployment checks. |
| `pkg/cryptoprice` | Read-only Polymarket web crypto reference-price client (`pkg/cryptoprice/client.go:1`). |
| `pkg/ctf` | Conditional Tokens Framework (CTF) helpers for position management. |
| `pkg/data` | Read-only Data API analytics client returning `pkg/types` DTOs. |
| `pkg/enabletrading` | Headless enable-trading flow: ClobAuth API keys and token approvals. |
| `pkg/funding` | Deposit-wallet funding helpers (ERC-20 transfers and balance checks). |
| `pkg/gamma` | Read-only Gamma API surface for embedded use. |
| `pkg/geoblock` | Polymarket geoblock verdict client (blocked flag plus caller IP/country/region). |
| `pkg/intel` | Read-only wallet intelligence DTOs and pure scoring helpers (`pkg/intel/types.go:1`, `pkg/intel/score.go:48`). |
| `pkg/marketdata` | Normalized live best bid, best ask, spread, midpoint, tick-size, last-trade, and book snapshots from public stream events. |
| `pkg/marketresolver` | Resolve market identifiers (ID, slug, token-id) to a canonical view. |
| `pkg/mcp` | Minimal read-only Model Context Protocol surface for agent integrations (`pkg/mcp/mcp.go:1`). |
| `pkg/openapi` | Minimal OpenAPI description for safe read-only tooling (`pkg/openapi/openapi.go:1`). |
| `pkg/orderbook` | Read-only CLOB order-book reader. |
| `pkg/orderfills` | On-chain `OrderFilled` truth models and readers (`pkg/orderfills/orderfills.go:1`, `pkg/orderfills/orderfills.go:50`). |
| `pkg/orderresults` | Order result types and response parsing for placement outcomes. |
| `pkg/pagination` | Cursor and offset pagination with concurrent batching. |
| `pkg/plugins` | Plugin interfaces for market data and risk extensibility. |
| `pkg/polyerrors` | Normalizes upstream Polymarket errors into stable kinds for operators, agents, and UI adapters (`pkg/polyerrors/polyerrors.go:1`). |
| `pkg/reconciliation` | Read-only operator report comparing CLOB order, Data API position/trade, relayer, and on-chain fill evidence (`pkg/reconciliation/reconciliation.go:1`). |
| `pkg/relayer` | Builder relayer primitives for wallet create and wallet batch flows. |
| `pkg/rfq` | Typed RFQ DTOs and validation; live submit is explicitly unsupported (`pkg/rfq/rfq.go:1`, `pkg/rfq/rfq.go:18`). |
| `pkg/rtds` | Real-Time Data Service WebSocket client for Chainlink oracle price ticks. |
| `pkg/settlement` | V2 winner redemption planning, adapter calls, and readiness gates. |
| `pkg/signers` | Public signing interfaces and safe local signer adapter (`pkg/signers/signers.go:1`, `pkg/signers/signers.go:21`). |
| `pkg/stream` | Read-only public CLOB WebSocket market stream client, including V2 custom feature events. |
| `pkg/types` | Public DTOs shared by SDK packages. |
| `pkg/universal` | Single client wrapping Gamma + CLOB + Data API + Discovery + Stream. |
| `pkg/upstreamdrift` | Checks saved official Polymarket docs indexes (`llms.txt`) against the Polygolem compatibility surface (`pkg/upstreamdrift/llms.go:1`). |
| `pkg/wallet` | Public deposit-wallet identity/readiness primitives — derive the POLY_1271 wallet and report wallet identity. |
| `pkg/experimental/orders` | **Experimental helper only** — fluent `OrderIntent` validation; stable user-directed order transactions live in `pkg/clob` and `polygolem clob create-order`. |
| `pkg/experimental/auth` | **Experimental** — EIP-712 domain helpers, signature type constants, and hex utilities (staged for SDK promotion). |

### Internal packages (`internal/`)

Implementation. Not part of the public SDK contract.

| Package | Purpose |
|---|---|
| `internal/auth` | L0/L1/L2 auth, EIP-712, deposit-wallet CREATE2 derivation, builder attribution, signers. |
| `internal/cli` | Cobra command construction and dependency wiring. |
| `internal/clob` | CLOB API client — full read + authenticated surface, EIP-712, POLY_1271, ERC-7739. |
| `internal/config` | Viper-backed config loading, defaults, environment binding, validation, redaction. |
| `internal/dataapi` | Data API client — positions, volume, leaderboards. |
| `internal/errors` | Structured error types and code helpers. |
| `internal/gamma` | Typed Gamma HTTP client — markets, events, search, tags, series, sports, comments, profiles. |
| `internal/marketdiscovery` | High-level market discovery service that combines Gamma and CLOB. |
| `internal/modes` | Read-only / paper / live mode parsing and gate checks. |
| `internal/output` | Stable table and JSON rendering plus structured errors. |
| `internal/paper` | Local-only paper positions, fills, and persisted state. |
| `internal/polytypes` | Polymarket protocol-level types shared across clients. |
| `internal/preflight` | Local and remote readiness checks. |
| `internal/relayer` | Builder relayer client — WALLET-CREATE, WALLET batch, nonce, polling. |
| `internal/risk` | Per-trade caps, daily loss limits, circuit breaker. |
| `internal/rpc` | Low-level Polygon RPC helpers: bytecode checks, direct transfers, swaps, deploy estimates. |
| `internal/stream` | WebSocket market stream implementation behind `pkg/stream`. |
| `internal/transport` | HTTP retry, rate limiter, circuit breaker, redaction. |
| `internal/wallet` | Deposit-wallet primitives — derive, deploy, status, batch signing. |

## Dependency direction

```text
cmd/polygolem
        |
internal/cli
        |
internal/{config, modes, preflight, output, errors}
        |
internal/{gamma, clob, dataapi, stream, relayer, rpc}   ← protocol clients
        |
internal/{auth, transport, polytypes}                   ← cross-cutting primitives
        |
internal/{wallet, orders, execution, risk, paper, marketdiscovery}
        |
pkg/{bridge, builder, clob, contracts, cryptoprice, ctf, data, enabletrading, funding, gamma, intel, marketdata, marketresolver, mcp, openapi, orderbook, orderfills, orderresults, pagination, plugins, relayer, rfq, rtds, settlement, signers, stream, types, universal, wallet}
pkg/experimental/{orders, auth}   ← experimental surfaces (staged for SDK promotion)
```

Command handlers parse flags, call package APIs, and render output via
`internal/output`. Protocol clients do not know about Cobra. Safety packages
do not depend on command text. Paper state stays local and never reaches
authenticated mutation endpoints.

Polygolem is not a bot or strategy engine. It exposes Polymarket API and
contract interfaces, with strong support for wallet setup, approvals, signing,
and user-directed order transactions. Callers decide markets, sides, prices,
sizes, timing, and whether to trade. See
[ADR-0002](adr/0002-polymarket-api-interface-boundary.md).

Cobra command handlers must not contain protocol or trading business logic.
That logic belongs in typed clients, application services, safety gates, and
paper-state packages where it is testable without executing the binary.

## Mode system

Mode selection starts in configuration and CLI flags, then flows through
`internal/modes` before command handlers call protocol clients or paper
state.

- **Read-only** (default): public market data only. May use
  `internal/gamma`, `internal/clob` (read endpoints), `internal/dataapi`,
  `internal/marketdiscovery`, and `internal/output`. Forbids signing or
  any mutation.
- **Paper**: local simulation. Combines read-only reference data with
  `internal/paper` state. Simulated actions stay local. Authenticated
  mutation APIs remain off-limits.
- **Live**: gated. Requires preflight + risk + funding gates to pass.
  Live execution operates through `internal/clob` (order build/sign + write
  endpoints), `internal/relayer`, `internal/rpc`, and `internal/wallet`. The
  default `polygolem` invocation does not enter live mode.

## Signature types

Polygolem supports **deposit wallet (POLY_1271 / type 3)** exclusively.
EOA, proxy, and Gnosis Safe are blocked by CLOB V2 and are not supported.

| Value | Status |
|-------|--------|
| `deposit` | ✅ Deposit wallet (POLY_1271). Required for all trading. |
| `eoa` | ❌ Blocked by CLOB V2 |
| `proxy` | ❌ Blocked by CLOB V2 |
| `safe` / `gnosis-safe` | ❌ Blocked by CLOB V2 |

Builder credentials are required for deposit wallet deployment via the
relayer. Order attribution uses the on-order `builder` bytes32 field (V2).

## Wallet contract capabilities

`pkg/contracts` is the stable source of Polygon addresses and approval sets:

- `TradingApprovals()` — six pUSD/CTF approvals for CLOB V2 order matching.
- `SettlementApprovals()` — four pUSD/CTF approvals for V2 split/merge/redeem adapters.
- `EnableTradingApprovals()` — two ERC-20 approvals mirrored from the polymarket.com Enable Trading flow.

`pkg/relayer` turns those metadata rows into deposit-wallet WALLET batch calldata.
`pkg/wallet` stays identity-only; deploy, approvals, funding, and settlement live
behind their own package seams.

## Public SDK boundary

`pkg/` is the public import boundary. Adding a package there is an SDK-level
commitment; keep unstable experiments under `pkg/experimental/` until their API
shape is proven.

Promoted DTO families cover Gamma/Data/CLOB market reads, account/order DTOs,
contract/readiness helpers, public/user streams, market data snapshots, crypto
reference prices, on-chain order fills, read-only agent surfaces, and signing
seams. Examples: `pkg/stream` exposes authenticated user-stream DTOs and a
`UserClient` (`pkg/stream/user.go:20`, `pkg/stream/user.go:36`), `pkg/rfq`
exposes DTO validation while live submit returns `ErrSubmitUnsupported`
(`pkg/rfq/rfq.go:16`, `pkg/rfq/rfq.go:114`), and `pkg/openapi`/`pkg/mcp` expose
read-only agent discovery surfaces (`pkg/openapi/openapi.go:14`,
`pkg/mcp/mcp.go:86`).

`pkg/experimental/` hosts APIs that are not yet SDK-stable. They follow the
same importable package rules as `pkg/`, but their APIs may change without a
major version bump. Packages graduate from `pkg/experimental/` to `pkg/` when
they have passed integration tests against live Polymarket for 30 days without
breaking changes.

## Safety boundaries

- Read-only is the default mode and is exercised by every public command.
- Paper mode never calls authenticated endpoints.
- Live commands require gates passing and builder credentials where
  applicable. Sigtype is hardcoded to 3 (POLY_1271, deposit wallet) — the
  only type Polymarket V2 accepts.
- Builder credentials and private keys are redacted by `internal/config`
  on every load.

## Update triggers

Refresh this page when:

- `find pkg -maxdepth 2 -type f -name '*.go'` shows a new or removed public package;
- a top-level command is added/removed in `internal/cli/root.go`;
- `pkg/contracts` approval sets change;
- any ADR changes the public SDK, bot/strategy boundary, or deposit-wallet-only model.

See [Architecture and Taxonomy Improvement Plan](ARCHITECTURE-TAXONOMY-PLAN.md)
for the next SDK naming and public-boundary cleanup work.
