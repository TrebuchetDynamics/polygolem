# polygolem TODO

## Packaging and trust fixes from skeptical review

These notes came from a blunt review of what looks wrong or fragile about the project. The goal is to make a skeptical operator trust the smallest possible safe path instead of adding more surface area.

### Public positioning

- [x] Narrow the homepage/README promise to one clear claim: **"Go CLI and SDK for safe Polymarket V2 deposit-wallet trading."** (README header, 2026-07-05; wording adjusted from the original "bot infrastructure" draft to respect the CONTEXT.md Polymarket API Interface avoid-list)
- [x] Replace broad marketing language such as "only production-ready option" with evidence-backed claims.
- [x] Add a visible "Known limitations" section near the top of the README.
- [x] Clearly distinguish what is proven from what is experimental, research, or internal archaeology.

### Trust and funds safety

- [x] Add a threat model covering private keys, signing, orders, funds, deposit wallets, API keys, and relayer credentials. (`docs/THREAT-MODEL.md`)
- [x] Add a funds-safety checklist before any live trading path. (`docs/THREAT-MODEL.md` assets/checklist, `docs/SAFE-HAPPY-PATH.md` stop conditions)
- [x] Document redaction guarantees and exactly what the CLI will never log or transmit. (`docs/SAFETY.md`, `docs/THREAT-MODEL.md`, `SKILL.md` redaction rules)
- [x] Add hard confirmations before live actions. (`--submit --confirm APPROVE_ADAPTERS` gates in `internal/cli/deposit_wallet.go`; `POLYGOLEM_MAX_LIVE_ORDER_USD` cap in `internal/cli/cmd_clob.go`)
- [x] Make safety disclaimers explicit enough that users understand stale markets, wrong token IDs, approvals, settlement assumptions, and command misuse can lose money. (`docs/THREAT-MODEL.md` token-ID/stale-market/approval rows, README "Known Limitations")

### Onboarding and demo path

- [x] Add one end-to-end demo path: read-only check → paper trade → live-readiness check → tiny live order with warnings. (`docs/SAFE-HAPPY-PATH.md`)
- [x] Make the deposit-wallet/new-user onboarding limitation explicit: one-time browser login is still required for some flows.
- [x] Clarify what is fully headless versus what requires browser/manual setup.
- [x] Provide a compatibility matrix for Go version, CLOB versioning, wallet type, signature type, relayer/deposit-wallet support, and supported flows. (`docs/COMPATIBILITY.md`, generated from the `pkg/capabilities` Capability Map by `cmd/polygolem_docs`; machine-readable twin at `docs/COMPATIBILITY.json`)

### Evidence and validation

- [x] Show proof points rather than claims: CI, test coverage, live smoke tests, compatibility matrix, known limitations, and failure cases. (All elements now exist: CI badge, CI-enforced 60% coverage floor in README Production Validation, `scripts/live-smoke.sh`, generated `docs/COMPATIBILITY.md`, README Known Limitations, and the failure-case docs under Research & Findings)
- [x] Add or document aggressive live smoke tests for Polymarket API/relayer behavior changes. (`scripts/live-smoke.sh`, `docs/UPSTREAM-DRIFT-RUNBOOK.md` weekly smoke)
- [x] Add a safe live-order walkthrough with exact preconditions and maximum spend caps. (`docs/LIVE-TRADE-WALKTHROUGH.md`; `POLYGOLEM_MAX_LIVE_ORDER_USD` cap)
- [x] Link at least one bot or paper-trading example that uses polygolem in practice. (`examples/boring-paper-bot/`, `examples/bot-basic/`, `examples/tradegate-bot/`)

### Scope control and docs cleanup

- [x] Separate the public reader path from internal archaeology: move obsolete investigations, correction notes, PRDs, probes, and historical blockers out of the main path. (`docs/README.md` canonical-vs-historical split; archaeology lives under `docs/history/`)
- [x] Keep `docs/history/BLOCKERS.md`/history useful but avoid making new users parse old wrong conclusions before they can run the happy path.
- [x] Re-evaluate whether Polydart, docs site, live probes, open-source project analysis, paper trading, SDK, CLI, and deposit-wallet flows should all be presented at once. (Decided 2026-07-05: the README front door presents only the polygolem core; sub-project and archaeology docs are indexed under a labeled "Sub-Projects & Internal Archaeology" shelf in `docs/README.md` rather than moved, preserving inbound links.)
- [x] Define the core thing polygolem does better than anything else: safe Polymarket V2 deposit-wallet (POLY_1271) trading from Go. (settled with the one-claim README promise, 2026-07-05)

### Dependency and version story

- [x] Revisit `go.mod` declaring `go 1.25.0`; explain or adjust if it weakens installability.
- [x] Explain the heavy crypto/zk/Ethereum indirect dependency tree so the "simple static binary" story remains credible. (`docs/DEPENDENCIES.md`)

## Blunt diagnosis

The project has value, especially the Polymarket V2/deposit-wallet knowledge encoded in Go, but the packaging is ahead of adoption proof. The weakest point is not code; it is trust. A stranger currently has little reason to run a young repo with funds. The next job is not more features; it is evidence, a smaller happy path, and operator confidence.

## Reconciliation note (2026-07-05)

Checked every unchecked item against the doc tree and code; items marked done above carry their evidence pointers. Still genuinely open, in priority order:

1. ~~Compatibility matrix~~ — delivered 2026-07-05: `docs/COMPATIBILITY.md` + `.json` generated from the `pkg/capabilities` Capability Map, exactly per the accepted generate-don't-hand-write approach.
2. ~~Published test-coverage numbers~~ — settled 2026-07-05: CI already measured and gated coverage; the floor is raised to 60% (62.9% actual at decision time) and documented in README Production Validation.
3. ~~Owner wording decisions~~ — all three settled 2026-07-05: one-claim promise, core-thing definition, and sub-project presentation (see checked items above).
