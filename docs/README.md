# Polygolem Documentation

This directory contains the canonical documentation for polygolem. For a single source of truth, always prefer the docs listed here. Older or superseded docs reference these canonical versions.

## Quick Start

| I want to... | Read this |
|-------------|-----------|
| Install and run polygolem | [../README.md](../README.md) |
| Understand the deposit wallet onboarding flow | [ONBOARDING.md](./ONBOARDING.md) |
| Use the browser fallback when headless login is blocked | [BROWSER-SETUP.md](./BROWSER-SETUP.md) |
| See a real EOA-to-filled-sell trade with every tx and gas figure | [LIVE-TRADE-WALKTHROUGH.md](./LIVE-TRADE-WALKTHROUGH.md) |
| See all CLI commands | [COMMANDS.md](./COMMANDS.md) |
| Understand the architecture | [ARCHITECTURE.md](./ARCHITECTURE.md) |
| Follow the smallest safe path from read-only to tiny live order | [SAFE-HAPPY-PATH.md](./SAFE-HAPPY-PATH.md) |
| Review safety and risk features | [SAFETY.md](./SAFETY.md) |
| Review the funds/credential threat model | [THREAT-MODEL.md](./THREAT-MODEL.md) |
| Understand the dependency/runtime story | [DEPENDENCIES.md](./DEPENDENCIES.md) |
| Use read-only MCP/OpenAPI agent surfaces | [MCP-OPENAPI.md](./MCP-OPENAPI.md) |
| Use the AI agent/SDK skill manifest | [../SKILL.md](../SKILL.md) |
| Run copyable starter examples | [../examples/](../examples/) |
| Run the opt-in live smoke script | [../scripts/live-smoke.sh](../scripts/live-smoke.sh) |
| Build or deploy the public docs website | [docs-site/](./docs-site/) |
| Review the open-source reinforcement roadmap | [POLYGOLEM-OPEN-SOURCE-REINFORCEMENT-PLAN.md](./POLYGOLEM-OPEN-SOURCE-REINFORCEMENT-PLAN.md) |

## Canonical Docs (Single Source of Truth)

### User Guides

| Doc | What It Covers | Status |
|-----|---------------|--------|
| [ONBOARDING.md](./ONBOARDING.md) | Complete automatic deposit wallet onboarding flow: derive, SIWE/profile/relayer auth, deploy, approve, enable trading, fund, trade. Polymarket login signs with the EOA; the deposit wallet remains the trading wallet. | **Canonical** |
| [BROWSER-SETUP.md](./BROWSER-SETUP.md) | Manual signing fallback and security guidance when `polygolem auth login` is blocked. | **Canonical** |
| [ENABLE-TRADING-HEADLESS.md](./ENABLE-TRADING-HEADLESS.md) | SDK flow for the UI Enable Trading typed-data prompts: ClobAuth API keys and token approvals. | **Canonical** |
| [COMMANDS.md](./COMMANDS.md) | Auto-generated CLI reference. Every command, flag, and example. | **Auto-generated** |
| [SAFE-HAPPY-PATH.md](./SAFE-HAPPY-PATH.md) | Minimal operator path: health, discovery, paper, readiness, tiny capped live order. | **Canonical** |
| [SAFETY.md](./SAFETY.md) | Read-only default, deposit wallet safety, risk breaker, circuit breaker. | **Canonical** |
| [THREAT-MODEL.md](./THREAT-MODEL.md) | Private key, API key, relayer key, approvals, token ID, and stale-market checklist. | **Canonical** |

### Technical Reference

| Doc | What It Covers | Status |
|-----|---------------|--------|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Package boundaries, dependency direction, design decisions. | **Canonical** |
| [CONTRACTS.md](./CONTRACTS.md) | All smart contract addresses, factory ABI, CREATE2 derivation. | **Canonical** |
| [POLY_1271-SIGNING.md](./POLY_1271-SIGNING.md) | How POLY_1271 / deposit wallet signing works. | **Canonical** |
| [POLYMARKET-COVERAGE-MATRIX.md](./POLYMARKET-COVERAGE-MATRIX.md) | Conservative SDK/CLI/docs/test coverage across Polymarket surfaces. | **Canonical** |
| [DEPENDENCIES.md](./DEPENDENCIES.md) | Why the binary has crypto-heavy Go dependencies while keeping no runtime sidecars. | **Canonical** |

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

*Last updated: 2026-05-11*
