# Polygolem Documentation

## What this is

Canonical docs index for polygolem, a Go Polymarket CLI/SDK. Prefer these docs
over historical plans; `docs/history/` preserves old audit context and may use
old command names.

## Start here

| Need | Read |
|---|---|
| Install and run | [../README.md](../README.md) |
| Command reference | [COMMANDS.md](./COMMANDS.md) |
| Compact source-backed wiki | [wiki/](./wiki/) |
| Safe onboarding | [ONBOARDING.md](./ONBOARDING.md) |
| Tiny live-order path | [SAFE-HAPPY-PATH.md](./SAFE-HAPPY-PATH.md) |
| Operator quickstart | [OPERATOR-ONE-PAGER.md](./OPERATOR-ONE-PAGER.md) |
| Browser/manual signing fallback | [BROWSER-SETUP.md](./BROWSER-SETUP.md) |
| Safety and threat model | [SAFETY.md](./SAFETY.md), [THREAT-MODEL.md](./THREAT-MODEL.md) |
| Real live trade reference | [LIVE-TRADE-WALKTHROUGH.md](./LIVE-TRADE-WALKTHROUGH.md) |
| Redeem incident/runbook | [DEPOSIT-WALLET-REDEEM-VALIDATION.md](./DEPOSIT-WALLET-REDEEM-VALIDATION.md) |

## Technical reference

| Doc | Purpose |
|---|---|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Package boundaries and dependency direction. |
| [adr/](./adr/) | Architecture decision records. |
| [CONTRACTS.md](./CONTRACTS.md) | Contract addresses, factory ABI, CREATE2 derivation. |
| [POLY_1271-SIGNING.md](./POLY_1271-SIGNING.md) | Deposit-wallet signing model. |
| [POLYMARKET-APIS.md](./POLYMARKET-APIS.md) | Official Polymarket APIs mapped to polygolem. |
| [POLYMARKET-COVERAGE-MATRIX.md](./POLYMARKET-COVERAGE-MATRIX.md) | SDK/CLI/docs/test coverage matrix. |
| [COMPATIBILITY.md](./COMPATIBILITY.md) | Generated compatibility contract from `pkg/capabilities`. |
| [DEPENDENCIES.md](./DEPENDENCIES.md) | Dependency/runtime rationale. |
| [UPSTREAM-DRIFT-RUNBOOK.md](./UPSTREAM-DRIFT-RUNBOOK.md) | Upstream API drift response. |

## Agent and SDK docs

| Doc | Purpose |
|---|---|
| [../SKILL.md](../SKILL.md) | Agent/SDK skill manifest. |
| [MCP-OPENAPI.md](./MCP-OPENAPI.md) | Read-only MCP/OpenAPI surfaces. |
| [JSON-CONTRACT.md](./JSON-CONTRACT.md) | CLI JSON envelope and error codes. |
| [POLYGOLEM-ROADMAP-MATRIX.md](./POLYGOLEM-ROADMAP-MATRIX.md) | Capability disposition roadmap. |
| [POLYGOLEM-OPEN-SOURCE-REINFORCEMENT-PLAN.md](./POLYGOLEM-OPEN-SOURCE-REINFORCEMENT-PLAN.md) | Open-source reinforcement plan. |

## Generated docs

Do not hand-edit generated output:

- [COMMANDS.md](./COMMANDS.md) and docs-site CLI reference come from
  `go run ./cmd/polygolem_docs`.
- [COMPATIBILITY.md](./COMPATIBILITY.md) and
  [COMPATIBILITY.json](./COMPATIBILITY.json) come from `pkg/capabilities`.
- `docs/docs-site/dist/` is build output.

## Historical docs policy

Top-level docs should be current and operator-usable. Deprecated drafts,
point-in-time audits, migration plans, and abandoned sub-project PRDs were
removed; use git history if archaeological detail is needed. Keep long-lived
historical evidence under [history/](./history/) only when active docs still cite
it.

## Update triggers

Refresh this index when commands, generated docs, public `pkg/` packages,
safety/onboarding behavior, or JSON-contract behavior changes.

*Last updated: 2026-07-07*
