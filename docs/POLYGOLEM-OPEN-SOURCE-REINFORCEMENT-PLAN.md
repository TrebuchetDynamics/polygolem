# Polygolem Open-Source Feature Reinforcement Plan

Date: 2026-06-10

Source comparison:

- `opensource-projects/FEATURE-MATRIX.md` — cross-repo feature inventory for Polymarket open-source projects.
- `docs/POLYMARKET-COVERAGE-MATRIX.md` — current polygolem SDK/CLI/docs/test surface.
- `docs/V2-PARITY.md` — focused V2 parity audit.
- Live source scan across `pkg/`, `internal/`, `cmd/`, and `tests/`.

## Executive summary

Polygolem is already stronger than most reference repos on the combination of Go SDK + single static CLI + V2 deposit-wallet safety. Against the open-source matrix, it should position itself closest to **`polymarket-go-sdk` for architecture**, **`Polymarket-golang` for V2 breadth**, and **`polymarket-kit`/`polymarket_cli` for AI/operator ergonomics**.

The best reinforcement path is not a broad rewrite. It is a set of narrow slices that make existing strengths harder to regress:

1. Add a published `polygolem` column/overlay to the matrix.
2. Close the few remaining feature/product gaps: Bridge withdrawals, RFQ, optional Turnkey/KMS signer adapters, and AI/MCP/OpenAPI tooling.
3. Reinforce architecture around signer boundaries, protocol conformance fixtures, typed runtime validation, and transport observability.
4. Convert reference repos into continuous drift detectors instead of one-time research notes.

## Execution progress

- Track A executed: `docs/POLYGOLEM-ROADMAP-MATRIX.md` gives polygolem a disposition for every capability in `opensource-projects/FEATURE-MATRIX.md`, enforced by `tests/open_source_roadmap_test.go`.
- Track B executed: `fixtures/protocol/README.md`, HMAC, ClobAuth, V2 POLY_1271 order, CTF calldata, and DepositWallet.Batch fixtures plus `tests/protocol_conformance_test.go` and `internal/clob/orders_protocol_fixture_test.go` pin CLOB L1/L2 auth, builder HMAC, V2 order typed-data/signature bytes, CTF calldata selectors, and DepositWallet.Batch typed-data hashes.
- Track C executed: `pkg/signers` publishes the stable public signer seam plus local signer adapter; `pkg/signers/http` adds an optional timeout-bound HTTP remote signer; `pkg/signers/kms` and `pkg/signers/turnkey` add provider-neutral custody backend seams without cloud/custody SDK dependencies; `tests/public_sdk_boundary_test.go` and `tests/repository_hygiene_test.go` now guard the package boundary.
- Track D executed for the planned safety-first surfaces: `pkg/bridge` exposes withdrawal/offramp dry-run types and an explicit unsupported-submit error; `pkg/rfq` publishes typed models, validation, and unsupported-submit guard; `pkg/ctf` exposes high-level split/merge dry-runs and readiness-gated submit-plan artifacts that still do not sign or submit.
- Track E executed: `fixtures/schemas/cli_envelope.schema.json` pins the CLI JSON envelope; RFQ, bridge withdrawal, and CTF operation request schemas pin new public DTO request shapes; `tests/json_schema_contract_test.go` validates envelope and schema coverage.
- Track F executed: `polygolem debug` emits redacted local diagnostics, endpoint configuration, and local preflight state in text or JSON; generated CLI docs are refreshed.
- AI/operator tooling executed: `pkg/mcp` and `cmd/polygolem_mcp` expose a read-only MCP manifest for safe health/discovery/data/orderbook/marketdata tools, reject mutating tool calls, support timeout-bound configured read-only tool handlers, and provide SDK client adapter wiring for health/search/positions/orderbook plus in-memory marketdata snapshots.
- OpenAPI executed: `pkg/openapi` and `cmd/polygolem_openapi` emit a minimal read-only OpenAPI 3.1 spec for health, diag, discovery, data positions, orderbook, and marketdata snapshots.
- Deployment examples executed: `docs/MCP-OPENAPI.md` documents read-only MCP/OpenAPI usage, embedding sketches, and excluded mutating surfaces.

## Polygolem vs matrix snapshot

Legend matches `opensource-projects/FEATURE-MATRIX.md`: `✅` first-class, `◐` partial/wrapper-level, `—` not a focus.

| Capability | Polygolem current state | Evidence | Reinforcement direction |
|---|---:|---|---|
| CLOB REST market data | ✅ | Coverage matrix lists order book, price, midpoint, spread, tick size, fee rate, last trade, price history, markets, token/condition lookup. | Keep as table-driven contract fixtures; add schema drift checks against reference payloads. |
| CLOB order placement/cancel/query | ✅ | V2 parity marks limit/market orders and cancel one/batch/all as SDK+CLI+test wired. | Add canonical golden order lifecycle fixtures and mock exchange conformance server. |
| V2 CTF Exchange order signing | ✅ | V2 parity defines V2 as CLOB Exchange V2 contracts, V2 order payload schema, and V2 relayer client. | Make the V2 signer package easier to audit with generated fixture docs and byte-for-byte comparisons to refs. |
| EIP-712 signing | ✅ | `internal/auth.Signer` exposes `SignTypedData`; deposit-wallet batch and order signing are wired. | Add optional external signer adapters while keeping local signing as default. |
| L2 HMAC/API-key auth | ✅ | Coverage matrix lists create/derive API keys and EOA-bound key derivation (owner address ignored in validated V2 path). | Add a dedicated auth conformance suite for timestamp/body canonicalization. |
| Builder attribution/auth headers | ✅ | `pkg/builder` provides local and remote builder-header signers; CLOB placement supports builder attribution. | Add builder-signing golden vectors copied from `go-builder-signing-sdk` and TS vendor refs. |
| RFQ | ◐ typed models only | `pkg/rfq` exposes request/quote/response DTOs, validation, unsupported-submit guard, schema fixture, and tests. | Add mock endpoint tests and authenticated calls only after Python/Rust payload capture. |
| WebSocket market/user streams | ✅ | Coverage matrix lists market stream and authenticated user stream; `pkg/stream.UserClient` is present. | Add chaos/reconnect integration tests and lifecycle metrics. |
| Gamma market/event metadata | ✅ | Coverage matrix lists Gamma markets, taxonomy, tags/categories/series/comments. | Add typed collections for opportunity scans from `polymarket-go-gamma-client`. |
| Data API / positions/activity | ✅ | Coverage matrix lists positions, closed positions, trades, activity, holders, value, markets traded, open interest, leaderboard, live volume. | Add all-market open-interest only after response shape is captured. |
| Bridge/deposits | ✅ deposits/quotes/status; ◐ withdrawal dry-run | `pkg/bridge` supports deposit address, supported assets, status, quote, withdrawal dry-run DTOs, schema fixture, and explicit unsupported-submit guard. | Add live withdrawal/offramp only after endpoint shape and safety constraints are verified. |
| CTF split/merge/redeem client ops | ✅ calldata + redeem flow; ◐ readiness-gated split/merge plan | `pkg/ctf` encodes split/merge/redeem calldata, high-level split/merge dry-runs, readiness-gated submit-plan artifacts, schema fixtures, and tests. | Add live submit helpers only behind explicit operator-confirm gates. |
| Collateral wrap/onramp/offramp | ◐ | Settlement/redeem is covered; bridge package is deposit/quote/status only. | Add pUSD wrap/offramp runbooks and explicit unsupported/offramp errors. |
| Rewards/rebates | ✅ | Universal client exposes reward config, earnings, reward percentages, user rewards, rebated fees, scoring, builder trades. | Add CLI/docs parity if any rewards endpoints are SDK-only. |
| Relayer/Safe support | ✅ deposit wallet relayer; Safe not target | Coverage matrix lists derive/deploy/status/nonce/batch/approve/fund/onboard/tx lookup for Polygon deposit wallet. | Keep Safe/EOA live trading unsupported unless product scope changes. |
| Turnkey wallet integration | ◐ provider-neutral signer seam | `pkg/signers/turnkey` exposes a Turnkey-style backend adapter without importing the Turnkey SDK. | Add provider-specific examples only behind optional dependencies/build tags after operator demand. |
| AWS KMS / remote signer | ◐ builder remote signer + HTTP/KMS-style signer seams | `pkg/builder.RemoteSigner` signs builder headers; `pkg/signers/http`, `pkg/signers/kms`, and `pkg/signers/turnkey` expose optional signer seams without default cloud/custody dependencies. | Add provider-specific examples only behind optional dependencies/build tags. |
| Proxy server / OpenAPI | ◐ read-only OpenAPI spec | `pkg/openapi` and `cmd/polygolem_openapi` emit a minimal OpenAPI 3.1 spec for safe read-only paths, documented in `docs/MCP-OPENAPI.md`. | Add a local proxy only after agent/operator demand is proven. |
| MCP server / AI tooling | ✅ read-only MCP manifest + SDK handler seam | `pkg/mcp` and `cmd/polygolem_mcp` expose safe read-only tools, reject mutating calls, support timeout-bound handlers, SDK client wiring, and marketdata snapshot reads. | Expand deployment examples as users adopt the surface. |
| CLI binary | ✅ | README positions polygolem as single static binary; coverage matrix maps every major surface to CLI commands. | Keep JSON envelope strict; add docs drift tests for every command. |
| Strong type/runtime validation | ✅ Go types; ◐ checked-in schemas | Go structs are broad; checked-in schemas now pin CLI envelope plus RFQ, bridge-withdraw, and CTF operation request DTOs with tests. | Add schemas for more public DTOs as they become compatibility-critical. |

## What to copy from each reference family

### 1. `polymarket-go-sdk`: architecture discipline

Keep copying its engineering posture, not its exact package names:

- execution lifecycle and idempotency for order posting;
- transport-wide retry/rate-limit/circuit-breaker policies;
- websocket lifecycle policy and state reporting;
- conformance-oriented tests around signing, retries, and errors.

Polygolem already has a transport client with retry, rate-limiter, circuit-breaker, and telemetry hooks. Reinforcement should focus on **making this policy observable and test-fixtured across every protocol client**, not adding a parallel client stack.

### 2. `Polymarket-golang`: V2 breadth

Use it as the breadth checklist for V2 order signing, bridge, Data/Gamma, rewards, websockets, and gasless operations. Polygolem already covers most of those surfaces. The remaining high-value breadth gaps are:

- RFQ;
- withdrawal/offramp;
- richer gasless CTF split/merge submit helpers;
- pUSD wrap/offramp docs and tests.

### 3. `ctf-exchange-v2`: protocol truth source

This should remain the source of truth for:

- V2 contract addresses and adapters;
- EIP-1271/deposit-wallet signing assumptions;
- collateral adapter behavior;
- split/merge/redeem calldata and settlement semantics.

Action: add a protocol conformance fixture folder containing contract-derived selectors, addresses, typed-data examples, and expected calldata.

### 4. `polymarket-kit`: operator and AI ergonomics

Copy selectively:

- OpenAPI/proxy ergonomics;
- MCP server read-only surfaces;
- explicit runtime schemas for JSON payloads.

Do not copy multi-runtime complexity into the core SDK. Keep Python/TS as interoperability targets, not dependencies.

### 5. `polymarket-go`: wallet ecosystem adapters

Turnkey is the only uniquely valuable surface here. Add it as an optional signer adapter after the signer seam is public and tested.

### 6. `go-builder-signing-sdk`: builder auth truth table

Use it for golden vectors. Polygolem already has local and remote builder-header signing, so the improvement is cross-repo conformance, not feature discovery.

### 7. `polymarket-go-gamma-client` and `polymarket_cli`: discovery UX

Copy their agent-friendly and analytics-friendly queries:

- wide-spread scanner;
- low-liquidity/high-volume scanner;
- new-market scanner;
- related-market/negative-risk candidates;
- soon-closing markets.

These fit naturally under `pkg/intel`, `discover`, or a new `scan` command group.

## Architecture reinforcement plan

### Track A — Matrix-backed public roadmap

**Goal:** make the comparison itself durable and keep it from going stale.

Tasks:

1. Add `polygolem` as a column or companion overlay in `opensource-projects/FEATURE-MATRIX.md`.
2. Add `docs/POLYGOLEM-ROADMAP-MATRIX.md` with three states per surface: current, next reinforcement, explicit non-goal.
3. Add a lightweight doc test that checks every capability row in the feature matrix has a polygolem disposition.

Acceptance criteria:

- Every capability in the open-source matrix has a polygolem status.
- The status cites source/doc/test evidence.
- Unsupported surfaces distinguish “not yet” from “not a product goal.”

### Track B — Protocol conformance fixtures

**Goal:** make V2 correctness provable against reference repos and contract truth.

Tasks:

1. Create `fixtures/protocol/` with EIP-712 order, CLOB auth, L2 HMAC, builder headers, CTF calldata, and deposit-wallet batch vectors.
2. Add tests that assert polygolem output matches the vectors byte-for-byte.
3. Import or hand-transcribe equivalent vectors from `go-builder-signing-sdk`, `Polymarket-golang`, TS CLOB V2, and `ctf-exchange-v2`.
4. Add a `go test ./tests/... -run Conformance` gate in CI.

Acceptance criteria:

- At least one golden vector per signing/auth/calldata family.
- Every fixture records source repo, source commit/path, and why it is trusted.
- Failure messages show exact byte/string diffs.

### Track C — Signer boundary hardening

**Goal:** match Rust/KMS/Turnkey-style flexibility without weakening the local-signing safety model.

Tasks:

1. Promote a stable public signer interface in `pkg/signers` that mirrors the internal typed-data needs.
2. Add adapters:
   - `pkg/signers/local` for private key;
   - `pkg/signers/http` for isolated signer service;
   - `pkg/signers/kms` behind build tags or optional module guidance;
   - later `pkg/signers/turnkey`.
3. Keep CLI private-key env workflow as the simplest default.
4. Add redaction and timeout tests for every remote signer path.

Acceptance criteria:

- Core CLOB and relayer flows accept a signer interface without requiring secrets to be strings.
- Remote signer failures are typed, redacted, and timeout-bound.
- No Turnkey/KMS dependency is pulled into the default binary unless explicitly enabled.

### Track D — Missing feature slices

Prioritized by feature-matrix leverage:

1. **Bridge withdrawal/offramp** — add only after endpoint payloads and custody risk are captured. Start with dry-run types and unsupported errors.
2. **RFQ** — add typed models and tests first, then authenticated calls.
3. **High-level CTF split/merge submit** — wrap existing calldata builders with deposit-wallet dry-run/submit helpers.
4. **Analytics scanners** — add Gamma/Data-powered discovery commands inspired by `polymarket-go-gamma-client`.
5. **Read-only MCP** — expose safe `health`, `discover`, `data`, `orderbook`, and `marketdata` tools; exclude live trading in v1.
6. **OpenAPI/local proxy** — generate from typed handlers only if MCP/agent users need HTTP rather than CLI.

Acceptance criteria:

- Each mutating feature ships with dry-run output, safety docs, and local mock tests before live instructions.
- Read-only agent features require no credentials by default.
- CLI JSON remains stable and documented.

### Track E — Runtime schema and JSON contract

**Goal:** approach `polymarket-kit` runtime confidence while staying Go-native.

Tasks:

1. Generate JSON Schema for public request/response DTOs from Go types.
2. Add fixture validation for CLI JSON envelopes.
3. Add schema examples to `docs/docs-site/`.
4. Add a compatibility test that prevents accidental field removals or envelope shape changes.

Acceptance criteria:

- Public CLI responses validate against checked-in schemas.
- Breaking JSON changes require an explicit fixture update.
- Docs examples are generated from the same fixtures.

### Track F — Observability and operations

**Goal:** become as operationally transparent as the best production SDKs.

Tasks:

1. Expose rate-limit status and circuit-breaker state in SDK and optional CLI diagnostics.
2. Add websocket lifecycle metrics: connected, reconnecting, last message time, dropped/deduped counts.
3. Standardize structured log fields across CLOB, Gamma, Data, Bridge, Relayer, and Stream clients.
4. Add `polygolem debug` commands for redacted config, endpoint reachability, version, and permissions.

Acceptance criteria:

- Operators can answer “is it us, Polymarket, the wallet, or rate limits?” from redacted output.
- No log line prints secrets or full auth headers.
- Transport tests assert retry/rate-limit/circuit-breaker telemetry events.

## Suggested milestone order

### Milestone 0 — Make the plan enforceable (1–2 days)

- Add matrix overlay with polygolem statuses.
- Add doc/test guard for matrix coverage.
- Create fixture folder layout and contributor instructions.

### Milestone 1 — Lock down correctness (3–5 days)

- Signing/auth/calldata conformance fixtures.
- Builder header cross-repo vectors.
- JSON envelope/schema fixtures.

### Milestone 2 — Add high-leverage missing surfaces (1–2 weeks)

- Bridge withdrawal dry-run/types.
- RFQ typed models and mock tests.
- CTF split/merge high-level dry-run/submit helpers.
- Analytics scanners.

### Milestone 3 — Architecture adapters (1–2 weeks)

- Public signer seam.
- HTTP/KMS/Turnkey optional adapters.
- Telemetry/rate-limit/status exposure.

### Milestone 4 — AI/operator layer (1 week)

- Read-only MCP server.
- Optional OpenAPI/local proxy.
- `diag` command group.

## Bug-hunt follow-up

Post-execution bug-hunt fixes added after the initial closeout:

- Bridge withdrawal dry-runs now reject zero, negative, decimal, or non-numeric `fromAmountBaseUnit` values before producing a preview.
- CTF split/merge dry-runs now reject malformed collateral addresses and malformed condition/parent collection IDs instead of silently zero-padding through go-ethereum helpers.
- OpenAPI now keeps `/orderbook/{token_id}` as a path parameter while preserving `/marketdata/snapshot?token_id=...` as a query parameter, with deterministic parameter ordering.
- RFQ validation and schema constraints now reject zero, negative, exponent, or malformed decimal amount strings before returning the unsupported-submit guard.
- MCP marketdata snapshots now require `token_id` consistently in the manifest and handler wrapper, matching the OpenAPI surface.
- The POLY_1271 docs and stale deposit-wallet-bound ClobAuth helper were corrected to the validated EOA-bound CLOB auth model, with safety tests blocking old ERC-7739 ClobAuth claims.
- Protocol fixtures now include `clob_auth.json` and `eip712_orders.json`, closing the original Track B gap for CLOB auth and V2 order typed-data/signature vectors.

## Execution closeout audit

The current execution pass is complete against the plan's safety-first scope:

- Every track A-F has at least one shipped code/doc/test artifact.
- Mutating new surfaces stop at dry-run, unsupported-submit, or readiness-gated plan artifacts; no new live funds-moving path was added.
- Open-source feature gaps called out in the plan now have durable roadmap dispositions in `docs/POLYGOLEM-ROADMAP-MATRIX.md`.
- Coverage docs and the docs index now reference the new RFQ, signer, CTF, MCP/OpenAPI, schema, and protocol fixture surfaces.
- Full verification passes with `go test ./...`.

Deferred future work is intentionally not part of this execution pass unless product scope changes or a follow-up issue is opened: provider-specific KMS/Turnkey examples, live RFQ/withdrawal endpoint implementation, a hosted/local HTTP proxy, and additional schema/fixture expansion as new DTOs become compatibility-critical.

## Non-goals unless product scope changes

- Default live EOA or Safe trading. Polygolem’s current deposit-wallet-only live-trading stance is a differentiator.
- Bundling Python/Node runtimes. Interop should happen through fixtures, schemas, CLI, MCP, and optional local proxy.
- Automatic live authenticated CI. Keep live trading manual and explicitly funded.
- Turnkey/KMS as required dependencies. They should be adapters.

## Verification evidence

Commands and files inspected while producing this plan:

- `opensource-projects/FEATURE-MATRIX.md`
- `docs/POLYMARKET-COVERAGE-MATRIX.md`
- `docs/V2-PARITY.md`
- `README.md`
- `pkg/bridge/bridge.go`
- `pkg/builder/builder.go`
- `internal/auth/auth.go`
- `internal/transport/client.go`
- `internal/clob/orders.go`
- `pkg/universal/client.go`
- `pkg/ctf/ctf.go`
- Source scan: `rg -n "SendHeartbeat|BuilderTrades|Scoring|PostOnly|Batch|CreateBatch|UserClient|Withdraw|Split|Merge|Redeem|RateLimit|slog|telemetry|cursor|next_cursor|OpenAPI|MCP|Turnkey|KMS|Remote" internal pkg cmd tests docs`
- Validation: `go test ./...` → `960 passed in 95 packages`
