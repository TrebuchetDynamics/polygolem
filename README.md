<h1 align="center">polygolem</h1>

<p align="center">
  <b>Production-safe Polymarket CLI and Go SDK</b>
</p>

<p align="center">
  Discover markets, paper trade, and trade on Polymarket V2 through deposit wallets.<br>
  One binary. No Python. No npm. No opaque wrappers.
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
- [Known Limitations](#known-limitations)
- [Try It — No Credentials Needed](#try-it--no-credentials-needed)
- [5-Minute Crypto Markets Demo](#5-minute-crypto-markets-demo)
- [Installation](#installation)
- [What's New](#whats-new)
- [Who This Is For](#who-this-is-for)
- [The Problem We Solve](#the-problem-we-solve)
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
# 1. List the current 5-minute markets for every supported asset
polygolem discover crypto-5m --enrich --json

# 2. Focus on the current BTC 5-minute window
polygolem discover crypto-window --asset BTC --interval 5m --enrich --json

# 3. Start with a clean paper account, then simulate an UP trade
polygolem paper reset --cash 100 --json
polygolem paper trade --asset BTC --interval 5m --side up --size 1 --json
polygolem paper positions --json
```

Look for `data.markets[].token_ids`, `outcomes`, `price`, and `spread` in the
JSON output. If a window is not available yet, wait for the next 5-minute UTC
boundary and retry. Full walkthrough: [POLYGOLEM-5M-CRYPTO-GUIDE.md](POLYGOLEM-5M-CRYPTO-GUIDE.md).

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
- **Bot developers** building automated trading strategies in Go
- **Quant developers** who want deterministic, compiled infrastructure with type safety
- **Operators** running headless trading systems that need auditability and local signing
- **Engineers** embedding Polymarket data and execution into larger Go services
- **Developers** who want one dependency, not a Python virtualenv + npm + Docker compose

If you are writing a Polymarket bot in Python or TypeScript, the [official CLOB clients](https://github.com/Polymarket/py-clob-client) are the right choice. If you are building in Go, or you want a single static binary with fixture-tested V2 deposit-wallet signing, polygolem focuses on that path with documented tests, fixtures, and live validation evidence.

---

## The Problem We Solve

Polymarket migrated to V2 in April 2026. The new model requires **deposit wallets** (ERC-1967 proxies with ERC-1271 validation) instead of EOAs as order makers. This broke most existing tooling.

| | Official Python/TS SDKs | polygolem |
|---|---|---|
| **Language** | Python / TypeScript | Go |
| **Dependencies** | pip/npm + 10+ transitive packages | Go stdlib + `cobra` |
| **Distribution** | Package manager install | Single static binary |
| **V2 deposit wallet** | Supported (with known bugs) | Supported, production-validated |
| **EOA signing** | Supported (produces ghost fills on V2) | **Blocked** — deposit-wallet only |
| **Version negotiation** | Hardcoded `CLOB_VERSION = "1"` → breaks on upgrades | Dynamic `/version` query before signing |
| **Credential security** | Auth headers leaked in error logs ([#327](https://github.com/Polymarket/clob-client/issues/327)) | Redacted in all output and logs |
| **Tick size caching** | In-memory per-instance, stale on update | Fresh fetch per order placement |
| **API key propagation** | 2-minute delay, no status polling | Derived on-demand with immediate use |
| **Local signing** | Optional (can use remote signers) | **Required** — key never leaves process |
| **External SDK in trust path** | Yes (Polymarket Python/TS SDKs) | No — all protocol code in this repo |
| **Go embedding** | Not possible | Native `pkg/` packages |
| **Read-only default** | No | Yes — credentials required explicitly |

**Concrete issues we avoid:**

- **Hardcoded `CLOB_VERSION = "1"`** in `py-clob-client` caused mass `order_version_mismatch` failures when Polymarket upgraded their EIP-712 domain in April 2026. Polygolem queries `/version` dynamically before every signing session.
- **Auth headers leaked in error logs** (TypeScript client [#327](https://github.com/Polymarket/clob-client/issues/327)). Polygolem redacts all secrets in errors, logs, and JSON output — tested and enforced.
- **Tick size caching bugs** ([#265](https://github.com/Polymarket/clob-client/issues/265)) cause valid orders to be rejected because stale tick sizes are cached per client instance. Polygolem fetches tick sizes fresh per order placement.
- **No official Go client exists** — only scattered community efforts with varying completeness. Polygolem is a unified, production-validated Go-native SDK.

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
- **V2 deposit wallet lifecycle** — Derive, deploy, fund, approve, trade — all headless
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
| [`pkg/stream`](pkg/stream) | Public CLOB WebSocket market stream |
| [`pkg/marketdata`](pkg/marketdata) | Live share-price snapshots from stream events |
| [`pkg/relayer`](pkg/relayer) | V2 Relayer client — WALLET-CREATE, batch, nonce |
| [`pkg/settlement`](pkg/settlement) | V2 winner redemption planning, adapter calls, readiness gates |
| [`pkg/bridge`](pkg/bridge) | Bridge deposits, status, quotes, and guarded withdrawal/offramp dry-runs |
| [`pkg/ctf`](pkg/ctf) | CTF split/merge/redeem calldata, high-level dry-runs, readiness-gated submit plans |
| [`pkg/rfq`](pkg/rfq) | Typed RFQ request/quote/response models with positive-decimal validation |
| [`pkg/signers`](pkg/signers) | Public signing seam with local, HTTP remote, KMS, and Turnkey adapters |
| [`pkg/orderbook`](pkg/orderbook) | Order book reader interface |
| [`pkg/builder`](pkg/builder) | Builder header signing — local EIP-712 and remote HTTP |
| [`pkg/enabletrading`](pkg/enabletrading) | Headless enable-trading: ClobAuth and token-approval typed-data signing |
| [`pkg/intel`](pkg/intel) | Wallet intelligence scoring — dossier alerts, shrinkage win rate, co-positioning signals |
| [`pkg/mcp`](pkg/mcp) | Read-only Model Context Protocol server and SDK handler wiring |
| [`pkg/openapi`](pkg/openapi) | Minimal read-only OpenAPI 3.1 spec generation |
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
# All 7 active 5m markets in one call
polygolem discover crypto-5m --enrich

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
| **Pre-trade caps + daily limits + circuit breaker** | Configurable in `internal/risk` |
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
| Check deposit wallet status | `polygolem deposit-wallet status` |
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
| [5-Minute Crypto Markets Guide](POLYGOLEM-5M-CRYPTO-GUIDE.md) | End-user demo for discovering and paper trading 5m crypto markets |
| [Safe Happy Path](docs/SAFE-HAPPY-PATH.md) | Smallest path from read-only checks to a tiny capped live order |
| [Live Trade Walkthrough](docs/LIVE-TRADE-WALKTHROUGH.md) | End-to-end reference run: every tx, gas figure, and pUSD movement |
| [Onboarding](docs/ONBOARDING.md) | Complete deposit wallet flow, troubleshooting |
| [Headless Enable Trading](docs/ENABLE-TRADING-HEADLESS.md) | SDK for UI ClobAuth and token-approval signing |
| [Browser Fallback](docs/BROWSER-SETUP.md) | Manual signing when headless login is blocked |
| [Safety](docs/SAFETY.md) | Risk controls, deposit-wallet-only enforcement |
| [Threat Model](docs/THREAT-MODEL.md) | Funds, credentials, approvals, wrong-token, and stale-market checklist |
| [Dependencies](docs/DEPENDENCIES.md) | Why crypto-heavy Go dependencies still ship as one binary |
| [Contracts](docs/CONTRACTS.md) | Contract addresses, factory ABI, CREATE2 derivation |
| [Architecture](docs/ARCHITECTURE.md) | Package boundaries and dependency direction |
| [Commands](docs/COMMANDS.md) | Auto-generated CLI reference |
| [Deposit Wallet Migration](docs/DEPOSIT-WALLET-MIGRATION.md) | V1→V2 survival guide |
| [polygolem.trebuchetdynamics.com](https://polygolem.trebuchetdynamics.com) | Searchable docs site |
| [SKILL.md](SKILL.md) | AI agent skill manifest — every command, env var, safety rule, and JSON contract |

---

## License

[MIT](LICENSE) © Trebuchet Dynamics
