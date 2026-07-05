<h1 align="center">polygolem</h1>

<p align="center">
  <b>Production-safe Polymarket CLI and Go SDK</b>
</p>

<p align="center">
  Interface with Polymarket APIs and contracts, including wallet setup and user-directed order transactions.<br>
  No bot decisions. No Python/npm runtime in the signing path. No opaque wrappers.
</p>

<p align="center">
  <a href="https://github.com/TrebuchetDynamics/polygolem/actions/workflows/ci.yml"><img src="https://github.com/TrebuchetDynamics/polygolem/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/TrebuchetDynamics/polygolem/releases"><img src="https://img.shields.io/github/v/tag/TrebuchetDynamics/polygolem?label=release&sort=semver" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/TrebuchetDynamics/polygolem" alt="Go Version"></a>
  <a href="https://goreportcard.com/report/github.com/TrebuchetDynamics/polygolem"><img src="https://goreportcard.com/badge/github.com/TrebuchetDynamics/polygolem" alt="Go Report Card"></a>
</p>

---

## Contents

- [Quick Start](#quick-start)
- [What is Polymarket?](#what-is-polymarket)
- [Known Limitations](#known-limitations)
- [Try It — No Credentials Needed](#try-it--no-credentials-needed)
- [5-Minute Crypto Markets Demo](#5-minute-crypto-markets-demo)
- [Installation](#installation)
- [What's New](#whats-new)
- [Who This Is For](#who-this-is-for)
- [Why Polygolem?](#why-polygolem)
- [Production Validation](#production-validation)
- [Features](#features)
- [Go SDK](#go-sdk)
- [Crypto Market Discovery](#crypto-market-discovery)
- [Safety Model](#safety-model)
- [Performance](#performance)
- [Common Workflows](#common-workflows)
- [The V2 Identity Model](#the-v2-identity-model)
- [Trade in Four Commands](#trade-in-four-commands)
- [Production Users](#production-users)
- [Contributing](#contributing)
- [Community](#community)
- [Docs](#docs)
- [License](#license)

---

## Quick Start

```bash
go install github.com/TrebuchetDynamics/polygolem/cmd/polygolem@latest

polygolem health
# {"clob":"ok","gamma":"ok"}
```

No credentials needed. Read-only is the default for everything until you set
`POLYMARKET_PRIVATE_KEY`. For a quick tour with zero setup, see
[Try It — No Credentials Needed](#try-it--no-credentials-needed) below.

---

## What is Polymarket?

Polymarket is a prediction-market exchange. Users trade YES/NO shares on
real-world outcomes such as elections, sports, crypto prices, and macro events.
A share usually trades between $0 and $1; the price is roughly the market's
implied probability before fees, spread, and liquidity caveats.

A useful Polymarket integration touches several surfaces:

| Surface | What it answers | Auth |
|---|---|---|
| Gamma API | What markets, events, tags, and comments exist? | None |
| CLOB API | What are the live books/prices, and how do I place/cancel orders? | None for reads, L2 auth for trading |
| Data API | What positions, holders, activity, and leaderboards exist? | Mostly none |
| WebSocket | What changed in real time? | None for market streams, L2 auth for user streams |
| Relayer + contracts | How do deposit-wallet deploys, approvals, settlement, and signatures work? | Relayer auth / wallet signatures |

Polygolem wraps these pieces in a Go CLI and SDK while keeping read-only paths
credential-free and mutating paths explicit. For the upstream view — what each
official service is for, its auth model, and where its documentation lives on
[docs.polymarket.com](https://docs.polymarket.com) — see
[docs/POLYMARKET-APIS.md](docs/POLYMARKET-APIS.md).

---

## Known Limitations

Polygolem is safest when used as a read-only CLI/SDK, paper-trading harness,
and deposit-wallet V2 implementation reference. Before funding a wallet, note:

- **Live trading can lose funds.** Use paper mode and preflight checks first;
  live mutation commands require explicit credentials and confirmations.
- **New-user deposit-wallet setup may require one-time browser login.** The CLI
  supports headless pieces, but Polymarket account/session state can still
  require manual browser setup. See [docs/ONBOARDING.md](docs/ONBOARDING.md).
- **Only deposit-wallet / POLY_1271 trading is supported.** EOA, proxy, and Safe
  trading modes are blocked for new production accounts.
- **Experimental SDK packages can change.** Anything under `pkg/experimental/`
  is not covered by the stable public SDK promise.
- **Read-only and fixture-tested flows are the strongest path.** Treat live
  relayer, CLOB, and chain behavior as upstream-dependent and verify with tiny
  capped runs.

---

## Try It — No Credentials Needed

You installed polygolem. Now try three things with zero setup:

```bash
# 1. Check API reachability (no credentials)
polygolem health --json

# 2. Search active markets
polygolem discover search --query "Will BTC" --limit 5 --json

# 3. Read a real order book
polygolem orderbook get --token-id 71321045679252249115448234976983616835904229510371422584850212744998471172014 --json
```

Every command accepts `--json` and returns a stable envelope: `{"ok": true, "version": "1", "data": ..., "meta": ...}`.
Full JSON contract at [docs/JSON-CONTRACT.md](docs/JSON-CONTRACT.md).

When you're ready to trade, see [Trade in Four Commands](#trade-in-four-commands).

---

## 5-Minute Crypto Markets Demo

This is the fastest end-user tour: find the live crypto up/down markets, inspect
one window, and paper trade it without connecting a wallet.

```bash
# 1. List current + next-hour 5-minute markets for every supported asset
polygolem discover crypto-5m --hours-ahead 1 --timezone America/Denver --json

# Optional: narrow to liquid majors and fetch live CLOB quote fields
polygolem discover crypto-5m --asset BTC --asset ETH --asset SOL --hours-ahead 1 --enrich --json

# 2. Focus on the current BTC 5-minute window
polygolem discover crypto-window --asset BTC --interval 5m --enrich --json

# 3. Start with a clean paper account, then simulate an UP trade
polygolem paper reset --cash 100 --json
polygolem paper trade --asset BTC --interval 5m --side up --size 1 --json
polygolem paper positions --json
```

Look for `data.markets[].token_ids`, `outcomes`, `liquidity_clob`, `best_bid`,
`best_ask`, and `book_spread` in the JSON output. `price` and `spread` are added
when `--enrich` succeeds. If a window is not available yet, wait for the next
5-minute UTC boundary and retry. Full walkthrough: [POLYGOLEM-5M-CRYPTO-GUIDE.md](POLYGOLEM-5M-CRYPTO-GUIDE.md).

---

## Installation

### go install (recommended)

```bash
go install github.com/TrebuchetDynamics/polygolem/cmd/polygolem@latest
```

### Build from source

```bash
git clone https://github.com/TrebuchetDynamics/polygolem
cd polygolem && go build -o polygolem ./cmd/polygolem
```

### Requirements

- Go 1.25+ (matches `go.mod` and CI)
- No runtime dependencies — single static binary

---

## What's New

- **Read-only MCP server** — expose health, discovery, data positions, orderbook, and marketdata snapshot tools through the Model Context Protocol for AI agent integration (`pkg/mcp`, `cmd/polygolem_mcp`)
- **Read-only OpenAPI spec** — emit a minimal OpenAPI 3.1 document for local proxy/tooling experiments (`pkg/openapi`, `cmd/polygolem_openapi`)
- **Public signer adapters** — stable `Signer` interface with local, HTTP remote, KMS-style, and Turnkey-style adapters (`pkg/signers`)
- **Polygolem diag** — redacted local diagnostics, endpoint configuration, and preflight state (`polygolem diag`)
- **Bridge withdrawal dry-run** — typed withdrawal DTOs, validation, and explicit unsupported-submit guard (`pkg/bridge`)
- **RFQ typed models** — request/quote/response DTOs, positive-decimal validation, and unsupported-submit guard (`pkg/rfq`)
- **CTF split/merge/redeem dry-runs** — high-level operation previews with readiness-gated submit-plan artifacts (`pkg/ctf`)
- **Protocol conformance fixtures** — golden vectors for CLOB auth, HMAC headers, V2 order EIP-712 hashes, CTF calldata, and deposit-wallet batch typed-data (`fixtures/protocol/`)
- **JSON schema fixtures** — checked-in schemas for CLI envelope, RFQ, bridge-withdrawal, and CTF operation requests (`fixtures/schemas/`)
- **Auth model correction** — CLOB L1/L2 auth confirmed EOA-bound; deposit-wallet identity belongs in POLY_1271 order fields, not ClobAuth headers

See [CHANGELOG.md](CHANGELOG.md) for full details.

---

## Who This Is For

- **End users** who want a credential-free CLI to inspect and paper trade fast crypto markets before funding an account
- **Bot or strategy developers** who need a Go interface into Polymarket APIs, not embedded strategy decisions
- **Quant developers** who want deterministic, compiled infrastructure with type safety
- **Operators** running user-directed headless transaction flows that need auditability and local signing
- **Engineers** embedding Polymarket data, wallet setup, and order transactions into larger Go services
- **Developers** who want one compiled artifact, not a Python virtualenv + npm + Docker compose

If you are writing a Polymarket bot in Python or TypeScript, start with
Polymarket's official clients. If you are building in Go, or you want a single
CLI binary with fixture-tested V2 deposit-wallet signing, Polygolem focuses on
that path.

---

## Why Polygolem?

Polymarket migrated to V2 in April 2026. The current production trading path
uses **deposit wallets** (ERC-1967 proxies with ERC-1271 validation) as order
makers, while the EOA still authenticates and signs. Polygolem exists to make
that model understandable, testable, and usable from Go.

Use Polymarket's official Python/TypeScript clients when those ecosystems fit
your stack. Use Polygolem when you need:

- **Go-native integration** — importable `pkg/` packages plus a CLI binary.
- **Read-only first workflows** — health, market search, order books, streams,
  public wallet analytics, and paper trading before credentials.
- **V2 deposit-wallet focus** — POLY_1271 order signing, relayer onboarding,
  approvals, balances, and settlement readiness documented in one place.
- **Fixture-backed protocol evidence** — checked-in vectors for CLOB auth,
  V2 orders, HMAC headers, CTF calldata, and deposit-wallet batches.
- **No Python/npm runtime in the signing path** — the production CLI and SDK are
  Go; Node is used only for the optional docs site/tooling.

### Evidence for these claims

| Claim | Where to verify |
|---|---|
| Official Python/TypeScript clients exist | `https://github.com/Polymarket/py-clob-client` and `https://github.com/Polymarket/clob-client` |
| Polygolem is Go-native | `go.mod`, `cmd/polygolem`, and `pkg/` packages |
| Read-only commands work without credentials | `polygolem health --json`, `polygolem discover search`, `polygolem orderbook get` |
| Deposit-wallet/POLY_1271 flow is fixture-tested | `fixtures/conformance/order_v2_poly1271.json`, `fixtures/protocol/eip712_orders.json`, `tests/conformance_vectors_test.go` |
| Secret redaction is tested | `internal/cli/cmd_diag_test.go` |
| Public API shapes are pinned | `fixtures/schemas/`, `tests/json_schema_contract_test.go` |

---

## Production Validation

> **Production-validated:** Polygon mainnet · 2026-05-11 reference run
>
> [Every tx hash, gas figure, and pUSD movement](docs/LIVE-TRADE-WALKTHROUGH.md)
> is documented from EOA private key to filled buy + sell.

Core trading flows validated today:

- Headless V2 relayer onboarding (SIWE + profile + relayer key mint)
- Deposit-wallet deploy + funding
- CLOB V2 order signing, placement, and cancellation
- Advanced order types (FOK, GTD, post-only)
- Market discovery, streaming, and paper trading

---

## Features

- **Market discovery** — Search, filter, and enrich Polymarket markets via Gamma + CLOB APIs
- **Deterministic crypto resolution** — Resolve current 5m/15m/1h/4h windows by slug (BTC, ETH, SOL, XRP, BNB, DOGE, HYPE)
- **Live market data** — Order books, prices, spreads, midpoints, tick sizes, last trades
- **WebSocket streaming** — Public CLOB market stream with auto-reconnect
- **V2 deposit wallet lifecycle** — Derive, deploy, fund, approve, and submit user-directed order transactions headlessly
- **Paper trading** — Simulate orders against live CLOB data with zero risk
- **Settlement readiness** — Check adapter approvals before redeeming winning positions
- **Local signing** — Private key never leaves the process; no external signing services
- **Secret redaction** — API keys and signatures are redacted in all output and logs
- **Read-only by default** — No credentials required for market data

---

## Go SDK

Every CLI subcommand is a thin wrapper around importable `pkg/` packages:

| Package | What it does |
|---|---|
| [`pkg/universal`](pkg/universal) | One typed client over Gamma + CLOB + Data API + Stream + Discovery (70+ methods) |
| [`pkg/clob`](pkg/clob) | CLOB V2 — market data, orders, balances, builder fees |
| [`pkg/gamma`](pkg/gamma) | Read-only Gamma market discovery (26 methods) |
| [`pkg/geoblock`](pkg/geoblock) | Geoblock verdict for the calling IP (blocked, country, region) |
| [`pkg/data`](pkg/data) | Read-only Data API analytics: positions, trades, holders, value, volume |
| [`pkg/stream`](pkg/stream) | Public CLOB WebSocket market stream |
| [`pkg/marketdata`](pkg/marketdata) | Live share-price snapshots from stream events |
| [`pkg/cryptoprice`](pkg/cryptoprice) | Read-only crypto reference prices for Up/Down resolution windows |
| [`pkg/wallet`](pkg/wallet) | Deposit-wallet identity/readiness — derive the POLY_1271 wallet |
| [`pkg/funding`](pkg/funding) | Explicit pUSD funding transfer helper for gated live flows |
| [`pkg/contracts`](pkg/contracts) | Polygon contract registry plus trading/settlement/enable-trading approval sets |
| [`pkg/relayer`](pkg/relayer) | V2 Relayer client — WALLET-CREATE, batch, nonce |
| [`pkg/settlement`](pkg/settlement) | V2 winner redemption planning, adapter calls, readiness gates |
| [`pkg/bridge`](pkg/bridge) | Bridge deposits, status, quotes, and guarded withdrawal/offramp dry-runs |
| [`pkg/ctf`](pkg/ctf) | CTF split/merge/redeem calldata, high-level dry-runs, readiness-gated submit plans |
| [`pkg/rfq`](pkg/rfq) | Typed RFQ request/quote/response models with positive-decimal validation |
| [`pkg/signers`](pkg/signers) | Public signing seam with local, HTTP remote, KMS, and Turnkey adapters |
| [`pkg/orderbook`](pkg/orderbook) | Order book reader interface |
| [`pkg/orderfills`](pkg/orderfills) | On-chain `OrderFilled` truth models and readers |
| [`pkg/orderresults`](pkg/orderresults) | Joined order/position/trade result reports |
| [`pkg/builder`](pkg/builder) | Builder header signing — local EIP-712 and remote HTTP |
| [`pkg/enabletrading`](pkg/enabletrading) | Headless enable-trading: ClobAuth and token-approval typed-data signing |
| [`pkg/intel`](pkg/intel) | Wallet intelligence scoring — dossier alerts, shrinkage win rate, co-positioning signals |
| [`pkg/mcp`](pkg/mcp) | Read-only Model Context Protocol server and SDK handler wiring |
| [`pkg/openapi`](pkg/openapi) | Minimal read-only OpenAPI 3.1 spec generation |
| [`pkg/pagination`](pkg/pagination) | Generic cursor, offset, and batch pagination helpers |
| [`pkg/plugins`](pkg/plugins) | Market-data and risk plugin interfaces for embedders |
| [`pkg/types`](pkg/types) | Shared public DTOs for SDK packages |
| [`pkg/marketresolver`](pkg/marketresolver) | Deterministic crypto window resolution (BTC/ETH/SOL/XRP/BNB/DOGE/HYPE) |

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
    "github.com/TrebuchetDynamics/polygolem/pkg/universal"
)

ctx := context.Background()
client := universal.NewClient(universal.Config{})

// Resolve current BTC 5m window
resolver := marketresolver.NewResolver("")
result := resolver.ResolveTokenIDsForWindow(ctx, "BTC", "5m", time.Now().UTC())
// result.Status = "available"
// result.UpTokenID = "208311606920..."
// result.DownTokenID = "988679547673..."

price, _ := client.Price(ctx, result.UpTokenID, "buy")
spread, _ := client.Spread(ctx, result.UpTokenID)
fmt.Printf("BTC 5m YES — price %s, spread %s\n", price, spread)
```

Full package boundaries in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## Crypto Market Discovery

Polymarket runs 5-minute up/down markets for major crypto assets. Polygolem
discovers them deterministically — no search index lag:

```bash
# Current + next-hour 5m markets in one call
polygolem discover crypto-5m --hours-ahead 1 --timezone America/Denver

# Liquid-major sweep only
polygolem discover crypto-5m --asset BTC --asset ETH --asset SOL --hours-ahead 1 --enrich

# Specific window
polygolem discover crypto-window --asset BTC --interval 5m

# Paper trade the current window in one step
polygolem paper trade --asset BTC --interval 5m --side up --size 1
```

Assets supported: BTC, ETH, SOL, XRP, BNB, DOGE, HYPE.

---

## Safety Model

| Guard | What it does |
|---|---|
| **Read-only by default** | No credentials = no authenticated operations |
| **Deposit-wallet only** | Cannot accidentally sign as EOA, proxy, or Safe |
| **Local signing** | Private key never leaves the process |
| **No external SDKs** | All wallet derivation, EIP-712, ERC-7739, and relayer code is in this repo |
| **Transaction safety caps + circuit breaker** | Guard user-directed transactions without selecting markets, sides, prices, or timing |
| **Secret redaction** | API keys and signatures are redacted in logs |

See [docs/SAFETY.md](docs/SAFETY.md) for the full model.

---

## Performance

Measured on Polygon mainnet during the 2026-05-11 reference run:

| Operation | Gas Cost (POL) | Paid By |
|---|---|---|
| Deposit wallet deploy (WALLET-CREATE) | ~0.20 POL | Polymarket relayer (sponsored) |
| Approval batch (6 calls) | ~0.12 POL | Polymarket relayer (sponsored) |
| CLOB order fill | ~0.05 POL | Polymarket matching engine (sponsored) |
| **User-paid total** | **~$0.01** | User (single pUSD funding transfer) |

All relayer and settlement gas is sponsored by Polymarket-run services. The user
pays only for the single ERC-20 transfer that funds the deposit wallet.

---

## Common Workflows

| I want to... | Run |
|---|---|
| Find an active market | `polygolem discover search --query "..."` |
| List all 5m crypto markets | `polygolem discover crypto-5m` |
| Inspect the book | `polygolem clob book <token-id>` |
| Derive deposit wallet | `polygolem deposit-wallet derive` |
| Onboard deposit wallet | `polygolem deposit-wallet onboard` |
| Check deposit wallet status | `polygolem deposit-wallet status --check-enable-trading` |
| Prepare by inspecting the exact book | `polygolem orderbook get --token-id <TOKEN_ID>` |
| Place a limit buy | `polygolem clob create-order --token <ID> --side buy --price 0.5 --size 10` |
| Place a market FOK buy | `polygolem clob market-order --token <ID> --side buy --amount 1 --price <cap>` |
| Cancel everything | `polygolem clob cancel-all` |
| Read collateral balance | `polygolem clob balance --asset-type collateral` |
| Paper trade | `polygolem paper trade --asset BTC --interval 5m --side up` |

Full CLI reference: [docs/COMMANDS.md](docs/COMMANDS.md).

---

## The V2 Identity Model

```
  EOA  ──signs──▶  Order
   │              (signatureType=3, maker=DepositWallet, signer=DepositWallet)
   │
   ▼ derives (CREATE2)              ▼ submitted by
 Deposit Wallet  ◀──holds pUSD──    Polymarket matching engine
 (ERC-1967 proxy,                   (gas-sponsored fillOrders settlement)
  validates signatures              ──┐
  via ERC-1271)                       │
                                      ▼
 V2 Relayer  ──sponsors──▶  WALLET-CREATE + approval batch
 (relayer-v2.polymarket.com)
```

Your EOA signs; the deposit wallet holds funds and is the on-order maker;
Polymarket-run services pay every gas fee except your single ERC-20 funding
transfer. See [the walkthrough](docs/LIVE-TRADE-WALKTHROUGH.md) for the full
lifecycle with real txes.

---

## Trade in Four Commands

```bash
export POLYMARKET_PRIVATE_KEY="0x..."

# One-command onboarding: auth + deploy + approve + fund
polygolem deposit-wallet onboard --fund-amount 0.71

# Sync CLOB balance
polygolem clob update-balance --asset-type collateral

# Place a market FOK buy
polygolem clob market-order \
  --token <ID> --side buy --amount 1 --price 0.012 --order-type FOK
# {
#   "success": true,
#   "orderID": "0x43083109...c423d793d",
#   "status": "matched",
#   "makingAmount": "1",
#   "takingAmount": "86.606666"
# }
```

After onboarding, every trade is fully headless. Total user-paid cost on the
reference run was **~$0.01 in POL gas** for the single ERC-20 transfer that
funds the deposit wallet.

> **Note:** Polymarket login signs with the EOA. `polygolem auth login` is still
> available as an explicit refresh/inspection command. Browser setup is
> fallback-only; see [docs/BROWSER-SETUP.md](docs/BROWSER-SETUP.md).

---

## Production Users

Polygolem is used in production by:

- **Trebuchet Dynamics** — institutional trading desk and quant research

*Want to be listed here? [Open an issue](https://github.com/TrebuchetDynamics/polygolem/issues) or reach out.*

---

## Contributing

Polygolem is a TDD-first project. All behavior changes land with tests, and new
tests fail before the implementation lands.

- **Bug reports:** [GitHub Issues](https://github.com/TrebuchetDynamics/polygolem/issues)
- **Feature requests:** [GitHub Issues](https://github.com/TrebuchetDynamics/polygolem/issues)
- **Security reports:** See [SECURITY.md](SECURITY.md) (do not file public issues)
- **Development guide:** See [CONTRIBUTING.md](CONTRIBUTING.md)

Build and test locally:

```bash
go build -o polygolem ./cmd/polygolem
go test ./...
go vet ./...
gofmt -w .
```

---

## Community

- **GitHub Discussions** — Q&A, show-and-tell, announcements
- **GitHub Issues** — Bug reports and feature requests
- **Documentation** — [polygolem.trebuchetdynamics.com](https://polygolem.trebuchetdynamics.com)

---

## Docs

| Document | What it covers |
|---|---|
| [Documentation Index](docs/README.md) | Canonical docs map and update triggers |
| [5-Minute Crypto Markets Guide](POLYGOLEM-5M-CRYPTO-GUIDE.md) | End-user demo for discovering and paper trading 5m crypto markets |
| [Operator One-Pager](docs/OPERATOR-ONE-PAGER.md) | Short no-wallet and pre-live checklist |
| [Safe Happy Path](docs/SAFE-HAPPY-PATH.md) | Smallest path from read-only checks to a tiny capped live order |
| [Live Trade Walkthrough](docs/LIVE-TRADE-WALKTHROUGH.md) | End-to-end reference run: every tx, gas figure, and pUSD movement |
| [Onboarding](docs/ONBOARDING.md) | Complete deposit wallet flow, troubleshooting |
| [Headless Enable Trading](docs/ENABLE-TRADING-HEADLESS.md) | SDK for UI ClobAuth and token-approval signing |
| [Browser Fallback](docs/BROWSER-SETUP.md) | Manual signing when headless login is blocked |
| [Safety](docs/SAFETY.md) | Risk controls, deposit-wallet-only enforcement |
| [Threat Model](docs/THREAT-MODEL.md) | Funds, credentials, approvals, wrong-token, and stale-market checklist |
| [Dependencies](docs/DEPENDENCIES.md) | Why crypto-heavy Go dependencies still ship as one binary |
| [Upstream Drift Runbook](docs/UPSTREAM-DRIFT-RUNBOOK.md) | Live smoke and fixture-first response plan for upstream API changes |
| [Contracts](docs/CONTRACTS.md) | Contract addresses, factory ABI, CREATE2 derivation |
| [Architecture](docs/ARCHITECTURE.md) | Package boundaries and dependency direction |
| [Commands](docs/COMMANDS.md) | Auto-generated CLI reference |
| [JSON Contract](docs/JSON-CONTRACT.md) | Stable `--json` success/error envelope |
| [MCP and OpenAPI](docs/MCP-OPENAPI.md) | Read-only agent/tooling integration surfaces |
| [Deposit Wallet Migration](docs/DEPOSIT-WALLET-MIGRATION.md) | V1→V2 survival guide |
| [polygolem.trebuchetdynamics.com](https://polygolem.trebuchetdynamics.com) | Searchable docs site |
| [SKILL.md](SKILL.md) | AI agent skill manifest — every command, env var, safety rule, and JSON contract |

---

## License

[MIT](LICENSE) © Trebuchet Dynamics
