# Polygolem Documentation

## What this is

This is the canonical docs index for polygolem: a Go Polymarket CLI/SDK (`go.mod:1`, `go.mod:3`) with a Cobra command tree (`internal/cli/root.go:58`) and stable JSON command envelopes (`internal/output/output.go:32`).

## Start here

Use this page to choose the right doc. Prefer the canonical docs listed below over historical plans. Historical notes under `docs/history/` preserve audit context and may contain old wording; generated site output under `docs/docs-site/dist/` is not edited by hand.

## Quick Start

| I want to... | Read this |
|-------------|-----------|
| Install and run polygolem | [../README.md](../README.md) |
| Understand the deposit wallet onboarding flow | [ONBOARDING.md](./ONBOARDING.md) |
| Use the browser fallback when headless login is blocked | [BROWSER-SETUP.md](./BROWSER-SETUP.md) |
| See a real EOA-to-filled-sell trade with every tx and gas figure | [LIVE-TRADE-WALKTHROUGH.md](./LIVE-TRADE-WALKTHROUGH.md) |
| See all CLI commands | [COMMANDS.md](./COMMANDS.md) |
| Understand the architecture | [ARCHITECTURE.md](./ARCHITECTURE.md) |
| Map Polymarket's official APIs (Gamma/CLOB/Data) to polygolem | [POLYMARKET-APIS.md](./POLYMARKET-APIS.md) |
| Follow the smallest safe path from read-only to tiny live order | [SAFE-HAPPY-PATH.md](./SAFE-HAPPY-PATH.md) |
| Read the one-page operator quickstart | [OPERATOR-ONE-PAGER.md](./OPERATOR-ONE-PAGER.md) |
| Review safety and risk features | [SAFETY.md](./SAFETY.md) |
| Review the funds/credential threat model | [THREAT-MODEL.md](./THREAT-MODEL.md) |
| Understand the dependency/runtime story | [DEPENDENCIES.md](./DEPENDENCIES.md) |
| Use read-only MCP/OpenAPI agent surfaces | [MCP-OPENAPI.md](./MCP-OPENAPI.md) |
| Use the AI agent/SDK skill manifest | [../SKILL.md](../SKILL.md) |
| Run copyable starter examples | [../examples/](../examples/) |
| Run the opt-in live smoke script | [../scripts/live-smoke.sh](../scripts/live-smoke.sh) |
| Investigate upstream API drift | [UPSTREAM-DRIFT-RUNBOOK.md](./UPSTREAM-DRIFT-RUNBOOK.md) |
| Build or deploy the public docs website | [docs-site/](./docs-site/) |
| Review the open-source reinforcement roadmap | [POLYGOLEM-OPEN-SOURCE-REINFORCEMENT-PLAN.md](./POLYGOLEM-OPEN-SOURCE-REINFORCEMENT-PLAN.md) |

## Source-backed orientation

| Area | Source of truth | Why it matters |
|---|---|---|
| CLI commands | `internal/cli/root.go:104`, `internal/cli/root.go:130` | Root command wires command groups and installs the JSON contract. |
| Command reference | `cmd/polygolem_docs/main.go:23`, `internal/cli/docs_generation.go:22` | `docs/COMMANDS.md` is generated from Cobra metadata; do not edit it by hand. |
| JSON envelope | `internal/cli/root.go:79`, `internal/output/output.go:54`, `internal/output/output.go:63` | Every command supports `--json`; success/error envelopes are rendered centrally. |
| CLI/package boundary | `internal/cli/doc.go:1`, `internal/cli/doc.go:4` | Command handlers should delegate to typed packages instead of holding protocol logic. |
| Runtime/deps | `go.mod:3`, `go.mod:8`, `go.mod:10` | Go version and core CLI/config dependencies live in the module manifest. |

## Canonical Docs (Single Source of Truth)

### User Guides

| Doc | What It Covers | Status |
|-----|---------------|--------|
| [ONBOARDING.md](./ONBOARDING.md) | Complete automatic deposit wallet onboarding flow: derive, SIWE/profile/relayer auth, deploy, approve, enable trading, fund, trade. Polymarket login signs with the EOA; the deposit wallet remains the trading wallet. | **Canonical** |
| [BROWSER-SETUP.md](./BROWSER-SETUP.md) | Manual signing fallback and security guidance when `polygolem auth login` is blocked. | **Canonical** |
| [ENABLE-TRADING-HEADLESS.md](./ENABLE-TRADING-HEADLESS.md) | SDK flow for the UI Enable Trading typed-data prompts: ClobAuth API keys and token approvals. | **Canonical** |
| [COMMANDS.md](./COMMANDS.md) | Auto-generated CLI reference. Every command, flag, and example. | **Auto-generated** |
| [OPERATOR-ONE-PAGER.md](./OPERATOR-ONE-PAGER.md) | One-page path for no-wallet checks and pre-live stop conditions. | **Canonical** |
| [SAFE-HAPPY-PATH.md](./SAFE-HAPPY-PATH.md) | Minimal operator path: health, discovery, paper, readiness, tiny capped live order. | **Canonical** |
| [SAFETY.md](./SAFETY.md) | Read-only default, deposit wallet safety, risk breaker, circuit breaker. | **Canonical** |
| [THREAT-MODEL.md](./THREAT-MODEL.md) | Private key, API key, relayer key, approvals, token ID, and stale-market checklist. | **Canonical** |

### Technical Reference

| Doc | What It Covers | Status |
|-----|---------------|--------|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Package boundaries, dependency direction, design decisions. | **Canonical** |
| [adr/](./adr/) | Architecture decision records, including the non-bot Polymarket API interface boundary. | **Canonical** |
| [CONTRACTS.md](./CONTRACTS.md) | All smart contract addresses, factory ABI, CREATE2 derivation. | **Canonical** |
| [POLY_1271-SIGNING.md](./POLY_1271-SIGNING.md) | How POLY_1271 / deposit wallet signing works. | **Canonical** |
| [POLYMARKET-APIS.md](./POLYMARKET-APIS.md) | Official Polymarket API architecture (Gamma, CLOB, Data API, Relayer, Bridge, WebSocket) with official-doc links and polygolem source citations. | **Canonical** |
| [POLYMARKET-COVERAGE-MATRIX.md](./POLYMARKET-COVERAGE-MATRIX.md) | Conservative SDK/CLI/docs/test coverage across Polymarket surfaces. | **Canonical** |
| [DEPENDENCIES.md](./DEPENDENCIES.md) | Why the binary has crypto-heavy Go dependencies while keeping no runtime sidecars. | **Canonical** |
| [UPSTREAM-DRIFT-RUNBOOK.md](./UPSTREAM-DRIFT-RUNBOOK.md) | Live smoke and fixture-first response plan for Gamma/CLOB/Data/Relayer drift. | **Canonical** |

### AI Agent Interface

| Doc | What It Covers | Status |
|-----|---------------|--------|
| [SKILL.md](../SKILL.md) | CLI skill manifest for agentic consumers — every command, env var, safety rule, and JSON contract. | **Canonical** |
| [MCP-OPENAPI.md](./MCP-OPENAPI.md) | Read-only MCP/OpenAPI deployment notes and excluded mutating surfaces. | **Canonical** |

### Planning

| Doc | What It Covers | Status |
|-----|---------------|--------|
| [POLYGOLEM-ROADMAP-MATRIX.md](./POLYGOLEM-ROADMAP-MATRIX.md) | Polygolem disposition for every capability in the open-source feature matrix. | **Planning** |
| [POLYGOLEM-OPEN-SOURCE-REINFORCEMENT-PLAN.md](./POLYGOLEM-OPEN-SOURCE-REINFORCEMENT-PLAN.md) | Executed reinforcement plan comparing polygolem to open-source Polymarket projects. | **Planning** |

### Research & Findings

| Doc | What It Covers | Status |
|-----|---------------|--------|
| [INTEGRATION_PLAN.md](../opensource-projects/INTEGRATION_PLAN.md) | Ecosystem survey, 7 gap implementations, headless onboarding blocker analysis. | **Canonical** |
| [LIVE-TRADING-BLOCKER-REPORT.md](./LIVE-TRADING-BLOCKER-REPORT.md) | Empirical live trading test results with real funds. | **Canonical** |
| [LIVE-TRADE-WALKTHROUGH.md](./LIVE-TRADE-WALKTHROUGH.md) | End-to-end 2026-05-08 reference run: every tx hash, gas figure, and pUSD movement from EOA private key to a filled buy + sell. | **Canonical** |
| [DEPOSIT-WALLET-REDEEM-VALIDATION.md](./DEPOSIT-WALLET-REDEEM-VALIDATION.md) | Scientific validation ladder and resolved live incident report for V2 settlement: official contracts, adapter readiness, redeem runbook, and deprecated fallback inventory. | **Canonical** |

## Update triggers

Refresh this index when:

- a top-level command is added/removed in `internal/cli/root.go`;
- `go run ./cmd/polygolem_docs` changes `docs/COMMANDS.md`;
- a public SDK package is added, moved, or promoted from `pkg/experimental/`;
- safety, onboarding, deposit-wallet, or JSON-contract behavior changes.

## Deleted Docs

These docs contained outdated or false claims and have been removed:

| Doc | Why It Was Deleted |
|-----|-------------------|
| `BUILDER-AUTO.md` | Superseded by `ONBOARDING.md` and the current EOA-login/deposit-wallet trading model. |
| `BUILDER-CREDENTIAL-ISSUANCE.md` | Superseded document with conflated credential types. |
| `DEPOSIT-WALLET-DEPLOYMENT.md` | Claimed "no browser needed" for full onboarding. Missed the deposit-wallet API key blocker. |

## How We Keep Docs Consistent

1. **Empirical testing over claims** — If a doc says something works, it must be backed by a test or live verification.
2. **One topic, one canonical doc** — Onboarding is only in `ONBOARDING.md`. Browser setup is only in `BROWSER-SETUP.md`.
3. **Cross-references, not duplication** — Docs link to canonical sources rather than repeating information.
4. **Deprecation notices** — Outdated docs get a clear banner at the top pointing to the replacement.

---

*Last updated: 2026-06-30*
