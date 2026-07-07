<h1 align="center">polygolem</h1>

<p align="center">
  <b>Go CLI and SDK for safe Polymarket V2 deposit-wallet trading</b>
</p>

<p align="center">
  Read markets, inspect books, simulate trades, onboard deposit wallets, and place user-directed orders.<br>
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

- [What Polygolem Is](#what-polygolem-is)
- [Install](#install)
- [Start Here by Audience](#start-here-by-audience)
- [No-Credentials Tour](#no-credentials-tour)
- [Command Map](#command-map)
- [Safe Trading Path](#safe-trading-path)
- [Go SDK](#go-sdk)
- [Safety Model](#safety-model)
- [Evidence and Validation](#evidence-and-validation)
- [Docs](#docs)
- [Contributing](#contributing)
- [License](#license)

---

## What Polygolem Is

Polymarket is a prediction-market exchange. Users trade YES/NO shares on real
outcomes; a share usually trades between `$0` and `$1`, roughly tracking market
probability before fees, spread, and liquidity caveats.

Polygolem is a Go interface to the surfaces a serious Polymarket integration
needs:

| Surface | What it answers | Auth |
|---|---|---|
| Gamma API | Markets, events, tags, comments | None |
| CLOB API | Books, prices, orders, balances | None for reads; L2 auth for trading |
| Data API | Positions, holders, activity, leaderboards | Mostly none |
| WebSocket | Live public market events and user streams | None for public streams; L2 auth for user streams |
| Relayer + contracts | Deposit-wallet deploys, approvals, settlement | Relayer auth / wallet signatures |

Polygolem keeps reads credential-free and makes every mutating path explicit.
It is not a bot or strategy engine: it never chooses markets, sides, sizes, or
risk for you.

---

## Install

```bash
go install github.com/TrebuchetDynamics/polygolem/cmd/polygolem@latest

polygolem ping
# {"clob":"ok","gamma":"ok"}
```

Build from source:

```bash
git clone https://github.com/TrebuchetDynamics/polygolem
cd polygolem
go build -o polygolem ./cmd/polygolem
```

Requirements:

- Go 1.25+ (see `go.mod` and `.github/workflows/ci.yml`)
- No Python/npm runtime for the CLI or signing path
- Node only if you build the documentation site

---

## Start Here by Audience

| You are... | Start with | Why |
|---|---|---|
| **New to Polymarket** | `polygolem ping`, then `polygolem markets search --query "Will BTC" --json` | Learn market/search/book shape without credentials |
| **Casual CLI user** | [No-Credentials Tour](#no-credentials-tour) | Inspect markets and simulate before funding anything |
| **Trader/operator** | [Safe Trading Path](#safe-trading-path), then `docs/SAFETY.md` | See the wallet, funding, approval, and cap gates before live orders |
| **Go SDK developer** | [Go SDK](#go-sdk), `pkg/universal`, `pkg/clob`, `pkg/gamma` | Import typed clients instead of shelling out |
| **Quant/research user** | `polygolem analytics`, `polygolem wallets`, `polygolem prices` | Pull public positions, flow, orderbook, and stream-derived signals |
| **AI/tooling builder** | `pkg/mcp`, `pkg/openapi`, `polygolem --json` | Stable JSON envelope plus read-only tool surfaces |
| **Contributor** | [Contributing](#contributing), `go test ./...`, `docs/COMMANDS.md` | Work from generated docs and validation gates |

If you use Python or TypeScript as your primary stack, Polymarket's official
clients may fit better. Use Polygolem when you need a Go SDK, one compiled CLI,
or a fixture-tested V2 deposit-wallet reference.

---

## No-Credentials Tour

These commands do not load a private key and do not require CLOB credentials.
Add `--json` for the stable `{ok, version, data, meta}` envelope.

```bash
# 1. API reachability
polygolem ping --json

# 2. Search active markets
polygolem markets search --query "Will BTC" --limit 5 --json

# 3. Read a book by token id
polygolem book get --token-id <TOKEN_ID> --json

# 4. Estimate fill/slippage without signing
polygolem exchange simulate --token <TOKEN_ID> --side buy --amount 10 --json

# 5. Simulate a crypto up/down trade with no wallet
polygolem sim reset --cash 100 --json
polygolem sim trade --asset BTC --interval 5m --side up --size 1 --json
```

For a focused crypto walkthrough, see
[POLYGOLEM-5M-CRYPTO-GUIDE.md](POLYGOLEM-5M-CRYPTO-GUIDE.md).

---

## Command Map

Top-level commands are grouped by safety posture in `polygolem --help`:

| Command | Audience | Purpose |
|---|---|---|
| `ping` | everyone | Check Gamma + CLOB reachability |
| `markets` | users, researchers | Search/list/enrich markets and crypto windows |
| `book` | traders, quants | Read books, midpoint, spread, tick size, fee rate |
| `exchange` | traders, operators | CLOB books, account reads, order placement/cancel, read-only simulation |
| `analytics` | researchers | Public Data API positions, trades, holders, volume, leaderboards |
| `wallets` | researchers | Read-only wallet dossiers, leaderboard signals, market flow |
| `prices` | quants, stream users | Live normalized market-data snapshots |
| `stream` | engineers | Raw/typed WebSocket market and user streams |
| `sim` | users, strategists | Local paper-trading simulation against public data |
| `wallet` | operators | Deposit-wallet derive/deploy/approve/fund/redeem lifecycle |
| `credentials` | operators | Auth readiness, SIWE login, export-key emergency path |
| `builder-keys` | operators | CLOB L2/builder credential helpers |
| `bridge` | operators | Bridge assets, quotes, deposit/status flows |
| `tx` | operators | Relayer transaction state |
| `risk` | operators | Advisory live posture status |
| `doctor`, `debug` | contributors, operators | Local readiness and redacted diagnostics |
| `check-upstream` | maintainers | docs.polymarket.com drift check |
| `events`, `version` | everyone | Event list and version output |

Full generated reference: [docs/COMMANDS.md](docs/COMMANDS.md).

---

## Safe Trading Path

Live trading can lose funds. The CLI supports it, but the default posture is
read-only until you provide credentials and explicit confirmations.

```bash
export SIGNER_PRIVATE_KEY="0x..."

# Optional explicit profile/login refresh. Polymarket login signs with the EOA.
polygolem credentials login

# One-command onboarding: auth + deploy + approve + fund.
polygolem wallet onboard --fund-amount 0.71 --confirm ONBOARD_WALLET

# Refresh CLOB balance/allowance.
polygolem exchange update-balance --asset-type collateral

# Place a tiny capped market/FOK buy.
POLYGOLEM_MAX_LIVE_ORDER_USD=1 polygolem exchange market-order \
  --token <TOKEN_ID> --side buy --amount 1 --price <WORST_ACCEPTABLE_PRICE>
```

Before running this path, read:

- [docs/SAFETY.md](docs/SAFETY.md) — guard model and live-money confirmations
- [docs/SAFE-HAPPY-PATH.md](docs/SAFE-HAPPY-PATH.md) — smallest live checklist
- [docs/LIVE-TRADE-WALKTHROUGH.md](docs/LIVE-TRADE-WALKTHROUGH.md) — reference run with tx hashes and gas figures
- [docs/ONBOARDING.md](docs/ONBOARDING.md) — full deposit-wallet flow
- [docs/BROWSER-SETUP.md](docs/BROWSER-SETUP.md) — fallback-only browser setup

---

## Go SDK

Every CLI command is a thin wrapper over importable Go packages. Use the SDK
when you want Polymarket data or user-directed flows inside a Go service.

| Need | Packages |
|---|---|
| One typed client | [`pkg/universal`](pkg/universal) |
| CLOB books, orders, balances | [`pkg/clob`](pkg/clob), [`pkg/orderbook`](pkg/orderbook), [`pkg/orderfills`](pkg/orderfills) |
| Market discovery | [`pkg/gamma`](pkg/gamma), [`pkg/marketresolver`](pkg/marketresolver), [`pkg/cryptoprice`](pkg/cryptoprice), [`pkg/geoblock`](pkg/geoblock) |
| Public analytics | [`pkg/data`](pkg/data), [`pkg/intel`](pkg/intel), [`pkg/orderresults`](pkg/orderresults), [`pkg/reconciliation`](pkg/reconciliation) |
| Streams and snapshots | [`pkg/stream`](pkg/stream), [`pkg/marketdata`](pkg/marketdata), [`pkg/rtds`](pkg/rtds) |
| Deposit wallet + contracts | [`pkg/wallet`](pkg/wallet), [`pkg/relayer`](pkg/relayer), [`pkg/contracts`](pkg/contracts), [`pkg/funding`](pkg/funding), [`pkg/settlement`](pkg/settlement), [`pkg/enabletrading`](pkg/enabletrading) |
| Signing and credentials | [`pkg/signers`](pkg/signers), [`pkg/builder`](pkg/builder) |
| Tooling surfaces | [`pkg/mcp`](pkg/mcp), [`pkg/openapi`](pkg/openapi), [`pkg/capabilities`](pkg/capabilities), [`pkg/compat`](pkg/compat), [`pkg/upstreamdrift`](pkg/upstreamdrift) |
| Error/pagination/types | [`pkg/polyerrors`](pkg/polyerrors), [`pkg/pagination`](pkg/pagination), [`pkg/types`](pkg/types) |
| Extension seams | [`pkg/plugins`](pkg/plugins), [`pkg/experimental`](pkg/experimental) |
| Dry-run transaction helpers | [`pkg/bridge`](pkg/bridge), [`pkg/ctf`](pkg/ctf), [`pkg/rfq`](pkg/rfq) |

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
    "github.com/TrebuchetDynamics/polygolem/pkg/universal"
)

func main() {
    ctx := context.Background()
    client := universal.NewClient(universal.Config{})

    resolver := marketresolver.NewResolver("")
    window := resolver.ResolveTokenIDsForWindow(ctx, "BTC", "5m", time.Now().UTC())

    price, _ := client.Price(ctx, window.UpTokenID, "buy")
    spread, _ := client.Spread(ctx, window.UpTokenID)
    fmt.Printf("BTC 5m YES — price %s, spread %s\n", price, spread)
}
```

Package boundaries and update triggers: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## Safety Model

| Guard | What it means |
|---|---|
| Read-only by default | Market/search/book/analytics commands need no credentials |
| Deposit-wallet-only trading | Production order maker is the POLY_1271 deposit wallet, not EOA/proxy/Safe |
| Local signing path | The CLI signs locally; no Python/npm runtime in the signing path |
| Live caps | `POLYGOLEM_MAX_LIVE_ORDER_USD` defaults to a tiny per-order cap |
| Typed confirmations | Live wallet commands require exact `--confirm` tokens |
| Secret redaction | Diagnostics redact API keys, signatures, and private-key material |
| SDK circuit breaker | SDK consumers can attach `TradeGate` / `internal/risk.Breaker`; CLI defaults remain explicit caps and confirmations |

Known limitations:

- Experimental SDK packages under `pkg/experimental/` can change.
- Upstream CLOB/relayer behavior can drift; use fixture-tested paths and tiny live trials.
- Browser setup is fallback-only, but upstream account/session state can still require manual recovery.

---

## Evidence and Validation

| Claim | Evidence |
|---|---|
| Go-native CLI/SDK | `go.mod`, `cmd/polygolem`, `pkg/` |
| Generated command docs are current | `docs/COMMANDS.md`, `internal/cli/docs_generation_test.go` |
| JSON envelope is pinned | `docs/JSON-CONTRACT.md`, `tests/json_schema_contract_test.go` |
| Protocol vectors are fixture-tested | `fixtures/protocol/`, `fixtures/conformance/`, `tests/conformance_vectors_test.go` |
| Read-only live coverage exists | `tests/e2e_*`, `scripts/live-smoke.sh`, `.github/workflows/smoke.yml` |
| CI validates Go code | `.github/workflows/ci.yml`: gofmt, `go vet`, `go test -short`, race, coverage |
| Docs site builds separately | `docs/docs-site/package.json`, `.github/workflows/docs-deploy.yml` |

Local validation:

```bash
go test ./...
go vet ./...
npm --prefix docs/docs-site run build  # optional docs-site check
```

---

## Docs

| Document | Start here when... |
|---|---|
| [docs/README.md](docs/README.md) | You want the docs map and update triggers |
| [docs/wiki/](docs/wiki/) | You want compact source-backed wiki pages for agents/contributors/operators |
| [docs/COMMANDS.md](docs/COMMANDS.md) | You need every CLI command/flag |
| [docs/SAFETY.md](docs/SAFETY.md) | You are anywhere near live funds |
| [docs/SAFE-HAPPY-PATH.md](docs/SAFE-HAPPY-PATH.md) | You want the smallest safe live path |
| [docs/ONBOARDING.md](docs/ONBOARDING.md) | You are setting up a deposit wallet |
| [docs/LIVE-TRADE-WALKTHROUGH.md](docs/LIVE-TRADE-WALKTHROUGH.md) | You need a production tx-by-tx reference |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | You are embedding or contributing to the SDK |
| [docs/POLYMARKET-APIS.md](docs/POLYMARKET-APIS.md) | You need upstream API boundaries |
| [docs/MCP-OPENAPI.md](docs/MCP-OPENAPI.md) | You are building agent/tool integrations |
| [docs/UPSTREAM-DRIFT-RUNBOOK.md](docs/UPSTREAM-DRIFT-RUNBOOK.md) | You maintain live read-only checks |
| [polygolem.trebuchetdynamics.com](https://polygolem.trebuchetdynamics.com) | You prefer searchable web docs |

---

## Contributing

Polygolem is TDD-first: behavior changes should land with focused tests.

```bash
go build -o polygolem ./cmd/polygolem
go test ./...
go vet ./...
gofmt -w .
```

- Bug reports and feature requests: [GitHub Issues](https://github.com/TrebuchetDynamics/polygolem/issues)
- Security reports: [SECURITY.md](SECURITY.md) — do not file public issues for private keys, signing paths, or credentials
- Contributor guide: [CONTRIBUTING.md](CONTRIBUTING.md)

---

## License

[MIT](LICENSE) © Trebuchet Dynamics
