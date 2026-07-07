# Changelog

All notable changes to `polygolem` are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.5.0] — 2026-07-07

### Changed

- **Breaking CLI rename:** top-level commands now use shorter names: `ping`,
  `markets`, `book`, `exchange`, `analytics`, `wallet`, `sim`, `prices`,
  `credentials`, `risk`, `doctor`, `debug`, `check-upstream`, `tx`, and
  `builder-keys`. The CLOB order simulator is now `exchange simulate`.
- **Docs cleanup:** removed deprecated drafts, point-in-time audit notes,
  abandoned sub-project PRDs, migration plans, and old blocker reports from
  top-level docs; current docs now point to the canonical onboarding, safety,
  wiki, and live-reference pages.
- **Capability map refresh:** generated compatibility docs now use the renamed
  CLI command paths.

## [v0.4.2] — 2026-07-07

### Added

- `polygolem clob simulate-order`: read-only CLOB book walk that estimates
  expected fill price, slippage, filled size, and unfilled amount without
  loading a private key, signing, or submitting an order.

## [v0.4.1] — 2026-07-06

Support for Polymarket's current wallet-approval flows (Get Paid Instantly
auto-redemption and the Combos-era Enable Trading batch), plus an RTDS
real-time price client.

### Added

- **Get Paid Instantly (auto-redeem).** New `deposit-wallet approve-auto-redeem`
  command submits the 3-call `setApprovalForAll` batch polymarket.com signs
  for the feature: CTF -> CtfAutoRedeem, CTF -> AutoRedeemer, and
  PositionManager -> AutoRedeemer. Once mined, winning positions are redeemed
  automatically after resolution. Dry-run by default; live submission
  requires `--submit --confirm APPROVE_AUTO_REDEEM` (covered by the
  fail-closed livegate test). SDK: `relayer.BuildAutoRedeemApprovalCalls`,
  `contracts.AutoRedeemApprovals`. Calldata is pinned byte-for-byte against
  a batch captured from the production UI.
- **Contract registry entries** for the new Polymarket contracts:
  `CtfAutoRedeem` (Sourcify-verified), `AutoRedeemer`, `PositionManager`
  (Combos ERC-1155), `CombosExchange`, and `CombosRouter` — verified against
  Polymarket's official deployment resources and on-chain bytecode.
- **RTDS WebSocket client** (`pkg/rtds`) for Chainlink crypto prices, with
  SDK reference docs.
- Automated Cloudflare Pages deploy for the docs site
  (`.github/workflows/docs-deploy.yml`).

### Changed

- **Enable Trading "Approve Tokens" batch grew from 2 to 6 calls**, matching
  the current polymarket.com UI: the original pUSD -> CTF and USDC.e ->
  CollateralOnramp approvals plus the Combos grants (pUSD approve and
  PositionManager `setApprovalForAll` for both CombosRouter and
  CombosExchange). `enabletrading.ValidateEnableTradingApprovalCalls` now
  validates the 6-call shape and rejects the old 2-call batch; wallets
  onboarded before this release should re-run `deposit-wallet
  enable-trading` once to add the Combos grants (idempotent).

## [v0.4.0] — 2026-07-06

Safety and usability release from a skeptical funds-safety review.

### Changed

- **Breaking:** deposit-wallet commands that submit real transactions now
  require a typed live-money confirmation token, matching the existing
  `approve-adapters`/`redeem` gate: `deposit-wallet batch` requires
  `--confirm SUBMIT_BATCH`, `deposit-wallet approve --submit` requires
  `--confirm APPROVE_TRADING`, and `deposit-wallet onboard` requires
  `--confirm ONBOARD_WALLET`. The token is checked before the private key is
  loaded. Scripts that call these commands must add the flag.

### Added

- Rich CLI help. `polygolem --help` now opens with a "what this is / start
  here" guide, worked examples, and a command list grouped by safety posture
  (read-only market data separated from the commands that move real funds).
  Every command group carries orientation text and a runnable example.
- Weekly read-only upstream smoke workflow (`.github/workflows/smoke.yml`):
  runs the credential-free smoke path plus `drift llms` on a schedule to catch
  Polymarket API/docs drift early.

### Fixed

- `clob batch-orders` enforces `POLYGOLEM_MAX_LIVE_ORDER_USD` per order and on
  the summed batch notional before signing (previously only `create-order` and
  `market-order` were capped).
- Rewrote `docs/SAFETY.md`, which described a Phase-1 state with no live
  execution; it now documents the real guard set. Corrected the README
  circuit-breaker claim: the breaker is an SDK-only `TradeGate`, not a CLI
  default.

### Internal

- Extracted `internal/livegate`, the single home for the live-order cap and the
  typed confirmation token, and routed every mutating command through it. Added
  a fail-closed coverage test that enumerates every live-money command and
  proves each rejects a violating invocation before the private key loads.

## [v0.3.0] — 2026-07-05

Cut to restore a correct `go install @latest` channel: `v0.2.1` predated the
sigtype-3 deposit-wallet derivation fix (factory switch to BEACON proxies), and
the invalid `v2026.5.9` tag was invisible to Go tooling. This is the first tag
that includes the derivation fix and the packages/commands below.

### Added

- Five public SDK packages: `pkg/capabilities` (the typed Capability Map —
  per-surface service, auth, wallet mode, read-only/mutating), `pkg/compat`
  (machine-readable compatibility contract), `pkg/polyerrors` (stable error
  kinds normalized from upstream failures), `pkg/reconciliation` (read-only
  evidence reconciliation report), and `pkg/upstreamdrift` (official `llms.txt`
  drift checker).
- `polygolem drift llms`: read-only, credential-free check that a saved official
  docs index still advertises every section the compatibility surface depends on.
- Generated `docs/COMPATIBILITY.md` and `docs/COMPATIBILITY.json` from the
  Capability Map, plus `docs/POLYMARKET-APIS.md` mapping the official Gamma/CLOB/
  Data APIs to polygolem.
- `pkg/geoblock`: read-only client for Polymarket's geoblock endpoint
  (blocked flag plus caller IP/country/region), migrated from the mega-bot
  consumer so all Polymarket HTTP lives in the SDK.
- `pkg/types`: executable top-of-book math on `CLOBOrderBook` — `BestBid`,
  `BestAsk`, and `AvailableAskSize(maxPrice)` — and `CLOBTickSize.Value()`
  for a parsed positive tick size, replacing per-consumer string parsing.
- `pkg/marketresolver`: canonical outcome helpers `NormalizeOutcome`,
  `OutcomeForToken`, and `CryptoMarket.OutcomeForToken` mapping a winning
  token ID to `up`/`down`/`unknown`.
- `pkg/marketresolver`: exported crypto-market parsing helpers migrated
  from the mega-bot consumer — `UpDownTokenIDs`, `InferTimeframe`,
  `InferTimeframeFromWindow`, `WindowFromSlug` (inverse of
  `CryptoWindowSlug`), `AssetSearchQueries`, `AssetMentioned`, and
  `ParseJSONStringList`.
- `pkg/contracts`: exported calldata builders for EOA and deposit-wallet
  flows — `ERC20{Approve,Transfer,Allowance,BalanceOf}Calldata`,
  `ERC1155{SetApprovalForAll,IsApprovedForAll}Calldata`,
  `Ramp{Wrap,Unwrap}Calldata` for the V2 collateral on/offramp pUSD
  conversion, plus `MaxUint256` and `Decode{Uint256,Bool}Result`.
  Selectors are keccak-verified in tests and byte-identical to the
  relayer's deposit-wallet batch encoding.

### Changed

- Authenticated commands now read `SIGNER_PRIVATE_KEY` first, falling back to
  the legacy `POLYMARKET_PRIVATE_KEY`. Docs, help text, and the generated
  command reference use the new name.
- Raised the CI statement-coverage floor from 50% to 60% and documented the
  enforced gate in the README.

### Fixed

- `clob batch-orders` now enforces `POLYGOLEM_MAX_LIVE_ORDER_USD` per order and
  on the summed batch notional before signing. Previously the cap guarded only
  `create-order` and `market-order`, so a batch could submit unlimited notional.
- Rewrote `docs/SAFETY.md`, which still described a Phase-1 state with no live
  execution and four enforced "live gates" that are actually advisory. It now
  documents the real guard set (read-only default, live-order cap, typed
  `APPROVE_ADAPTERS`/`REDEEM_WINNERS` confirmation tokens, credential handling).

## [v0.2.1] — 2026-06-30

### Added

- **Raw WebSocket recorder support.** Market streams now expose deduplicated raw payload callbacks and per-event stats for collector freshness receipts.
- **Crypto stream lookahead refresh.** `stream crypto` subscribes to current and near-future window tokens and refreshes at interval boundaries.
- **Polymarket interface ADRs.** Documented the non-bot API boundary, deposit-wallet-only trading path, and stable public SDK surface.

### Changed

- **Docs now describe user-directed transactions** instead of embedded trading decisions, keeping strategy choices outside Polygolem.

## [v0.2.0] — 2026-06-22

### Added

- **Wallet Intelligence V1 (`pkg/intel`, CLI workflows, and context glossary).**
  Adds reproducible wallet dossier scoring, formula-versioned score output,
  batch dossier alerts, source-authority/conflict handling, and E2E coverage
  for read-only wallet intelligence signals.
- **Public protocol conformance artifacts.** Added fixtures and tests for
  CLOB auth, builder headers, POLY_1271 order signatures, deposit-wallet
  batches, CTF calldata, JSON envelopes, and schema contracts so downstream
  SDKs can verify parity without live credentials.
- **MCP and OpenAPI surfaces.** Added `cmd/polygolem_mcp`,
  `cmd/polygolem_openapi`, `pkg/mcp`, `pkg/openapi`, and documentation for
  read-only agent/tool integrations.
- **Public SDK expansion.** Added order fills, RFQ dry-run DTOs, CTF operation
  DTOs, signer interfaces plus HTTP/KMS/Turnkey signer adapters, stream user
  helpers, richer Gamma discovery collections, and more V2 market/type helpers.
- **Examples and operator docs.** Added read-only monitor scripts, paper
  strategy examples, basic/tradegate/boring-paper bots, operator one-pager,
  safe happy path, threat model, upstream drift runbook, and coverage/parity
  matrices.
- **Live BTC 5-minute CLOB contract test and live-smoke script.** These expand
  optional live validation while keeping default CI on fixture-backed `-short`
  tests.

### Changed

- **CLOB live order paths are guardable through `TradeGate`.** `pkg/clob` and
  `pkg/universal` can thread an opt-in gate through create-order paths while
  keeping cancel paths open for risk reduction.
- **Paper sell accounting now uses average-cost realized PnL.** Paper account
  sells route through `State.Sell`, update remaining cost basis, and report
  realized PnL deterministically.
- **README and docs were reworked for open-source users.** The README now
  emphasizes no-credential read-only flows, known limitations, production-safe
  deposit-wallet usage, and SDK/package boundaries.
- **CI now runs short tests and a race-detector pass.** Live-network E2E tests
  no longer gate pull requests, while concurrency regressions are covered by
  `go test -race -short ./...`.
- **Dependency updates.** Bumped `github.com/ethereum/go-ethereum` to v1.17.3,
  `github.com/spf13/cobra` to v1.10.2, and `golang.org/x/crypto` to v0.53.0.
- **Docs site moved under `docs/docs-site/`.** Historical plans were archived
  under `docs/history/` and the public documentation map was refreshed.

### Fixed

- **CLOB market orders price from the best book level** instead of stale or
  inappropriate pricing inputs.
- **Silent stream connections are detected and reconnect budget is reset** so
  dead WebSocket sessions recover more reliably.
- **Compact JSON signing is escape-aware** for HMAC body signing.
- **Transport retries close response bodies on every iteration** to avoid
  leaking resources.
- **Risk breaker stays halted while still over loss/position limits** and clamps
  negative loss records.
- **Gamma query serialization now includes all declared filters.**
- **Batch price validation, comment URL encoding, dedup caps, and TZ-offset
  parsing** were hardened.
- **Marketdata limit handling no longer truncates unset limits to one**, and
  JSON output buffering prevents partial output on errors.
- **Pagination helpers and relayer padding reject bad input safely.**
- **Wallet nil dereference, mode validation, deployment-source labels, paper
  sell-accounting edge cases, and AutoHeartbeat test races** were fixed.
- **Swap submission now confirms receipts** so reverted swaps are not reported
  as success.

### Removed

- Removed unenforced risk policy fields (`MaxOrderUSD`, `MaxOpenOrders`) that
  looked authoritative but were not applied.

## [v0.1.1] — 2026-05-11

### Added

- **Crypto market discovery commands.**
  - `polygolem discover crypto` — search active crypto markets by asset and
    interval (5m, 15m, 1h, 4h) with optional CLOB price/spread enrichment.
  - `polygolem discover crypto-window` — deterministic slug resolution for
    the current time-windowed market (`btc-updown-5m-<unix>`). Bypasses search
    index lag; hits the exact window directly.
  - `polygolem discover crypto-5m` — resolves all 7 active 5-minute crypto
    markets (BTC, ETH, SOL, XRP, BNB, DOGE, HYPE) in a single call with
    consolidated token IDs, condition IDs, and optional live prices.
- **Paper trading commands.**
  - `polygolem paper buy` / `polygolem paper sell` — simulate orders against
    live CLOB best ask/bid with local $10,000 starting cash.
  - `polygolem paper positions` / `polygolem paper reset` — inspect and wipe
    paper state.
  - `polygolem paper trade` — one-command workflow: resolve current window,
    fetch live price, and execute paper trade in a single step.
  - `polygolem paper crypto` — discover crypto markets and return token IDs
    ready for paper trading.
- **`CryptoWindowSlug` exported from `pkg/marketresolver`.** Deterministic
  slug generator for downstream consumers (bots, dashboards) that need to
  construct Polymarket crypto event slugs without hitting search.
- **V2 settlement gate (`pkg/settlement`, `polygolem deposit-wallet settlement-status`).**
  Read-only readiness check: deposit-wallet bytecode, relayer credentials,
  Data API positions reachability, and CTF approvals for both V2 collateral
  adapters.
- **Live E2E demo tests.** `tests/e2e_live_market_extraction_test.go`,
  `e2e_multi_market_stress_test.go`, `e2e_polygolem_demo_test.go` —
  production-validated read-only flows against live Polymarket APIs.
- **Coverage gate at 60%** with baseline tracking in CI.
- **Property-based tests** (`testing/quick`) for critical path validation.

### Changed

- **CLI root.go split** from 1,677 lines into 10 domain-specific files
  (`cmd_discover.go`, `cmd_clob.go`, `cmd_paper.go`, etc.).
- **Deposit-wallet redeem docs and CLI help** now hard-disable fallback
  thinking: V2 settlement is relayer + collateral adapter only, with no
  direct EOA, raw CTF, SAFE, or PROXY route.
- **`RELAYER_ALLOWLIST_BLOCKED`** now tells operators to verify the local
  contract registry against Polymarket's current contract reference before
  escalating. The stale upstream issue tracker is no longer surfaced as a
  current source of truth.
- **`OnboardDepositWallet` ships a 10-call post-deploy batch** (6 trading +
  4 adapter approvals) so new wallets are redeem-ready out of the box.
- **`ResolveTokenIDsAt` fails closed on slug-hit window mismatch** instead of
  silently substituting a different window.

### Fixed

- **7 Go-specific bugs** identified and fixed:
  - `hexToBytes`/`hexDecodeInto` now use `encoding/hex` with panic on invalid
    hex instead of manual parsing.
  - `buildOrderTypedData` now returns `(apitypes.TypedData, error)` with
    proper error handling.
  - WebSocket race condition fixed with `mc.mu` protecting `mc.conn`.
  - Deposit-wallet deploy false-negative: `eth_getCode` is now the source of
    truth when the relayer reports `STATE_FAILED`.
  - Position schema decode: JSON tags corrected from snake_case to camelCase
    to match live Polymarket Data API.
  - `CtfCollateralAdapter` and `NegRiskCtfCollateralAdapter` updated to
    current official Polygon addresses.
  - CLOB market buy rounding aligned with V2 expectations.

### Removed

- Removed the deprecated `deposit-wallet deploy-onchain` command and internal
  direct EOA factory deploy helper. The production deposit-wallet factory gates
  `deploy(...)` and `proxy(...)` behind `onlyOperator`, so the relayer
  `WALLET-CREATE` path is the only supported Polygolem deploy surface.
- Removed deprecated `pkg/bookreader` in favor of `pkg/orderbook`.

## [v2026.5.9] — 2026-05-09

Release version: `v0.1.0`.

### Added

- **Market-window guard (`pkg/marketresolver`).** New strict
  `ResolveTokenIDsForWindow(asset, timeframe, windowStart)` returns
  `StatusAvailable` only when the matched market's `startDate` exactly
  equals `windowStart`. New `StatusWindowMismatch` distinguishes
  wrong-window from no-market — intended as the only resolver entry
  point on the live order-placement path. `CryptoMarket` and
  `ResolveResult` now carry `StartDate`/`EndDate`.
- **V2 collateral adapter registry (`pkg/contracts`).**
  `CtfCollateralAdapter`, `NegRiskCtfCollateralAdapter`,
  `CollateralOnramp`, `CollateralOfframp`, and `PermissionedRamp`
  constants and registry fields. New helper
  `RedeemAdapterFor(negRisk bool) string` selects the right adapter
  for a market kind.
- **V2 redeem SDK (`pkg/settlement`).** `FindRedeemable`,
  `BuildRedeemCall`, `SubmitRedeem`, and the `RedeemablePosition` /
  `RedeemResult` DTOs. Calldata reuses `pkg/ctf.RedeemPositionsData`;
  the adapter ignores `collateralToken`, `parentCollectionId`, and
  `indexSets` (uses `CTFHelpers.partition()=[1,2]` internally), so we
  pass zero values. `SubmitRedeem` dedupes by `conditionID` (collapses
  YES/NO splits) and caps batches at `DefaultBatchLimit=10`.
- **Adapter approval primitives (`pkg/relayer`, `internal/relayer`).**
  `BuildAdapterApprovalCalls()` returns the 4 calls a deposit wallet
  must submit before V2 split/merge/redeem (pUSD `approve` + CTF
  `setApprovalForAll` for both V2 collateral adapters). Idempotent.
- **Operator CLI surface.**
  - `polygolem deposit-wallet approve-adapters` — one-shot migration
    for existing wallets. Dry-run by default; `--submit` requires
    `--confirm APPROVE_ADAPTERS`.
  - `polygolem deposit-wallet redeemable` — read-only list of
    redeemable positions for the deposit wallet.
  - `polygolem deposit-wallet redeem [--limit N] [--rpc-url URL]
    [--submit --confirm REDEEM_WINNERS]` — pre-checks
    `CTF.isApprovedForAll(wallet, adapter)` and refuses to sign with a
    structured pointer to `approve-adapters` if any approval is
    missing.
- **Adapter-approval pre-check (`internal/rpc.IsApprovedForAll`).**
  ERC-1155 approval check via `eth_call` (selector `0xe985e9c5`),
  used by the redeem CLI to fail-closed before signing.
- **Position V2 fields.** `pkg/types.Position` and
  `internal/dataapi.Position` now surface `Redeemable`, `Mergeable`,
  `NegativeRisk`, `Outcome`, `OutcomeIndex`, `OppositeOutcome`,
  `OppositeAsset`, `EndDate`, `Title`, `Slug`, `EventSlug`, `Icon`,
  `EventID`, `ProxyWallet`, `InitialValue`, `CurrentValue`,
  `TotalBought`, `RealizedPnl`, `PercentRealized`, `CashPnl`,
  `PercentPnl`.

### Changed

- **`OnboardDepositWallet` ships a 10-call post-deploy batch** (6
  trading + 4 adapter approvals) so new wallets are redeem-ready out
  of the box. Existing live wallets must run
  `deposit-wallet approve-adapters` once.
- **`ResolveTokenIDsAt` fails closed on slug-hit window mismatch**
  instead of silently substituting a different window.

### Fixed

- **Position schema decode bug.** `Position` JSON tags were
  snake_case (`token_id`, `condition_id`, `avg_price`, `unrealized_pnl`,
  `side`, `market_id`) but the live Polymarket Data API returns
  camelCase (`asset`, `conditionId`, `avgPrice`, `cashPnl`, no
  `side`/`market_id` at all). Existing tests round-tripped through
  their own snake_case fixtures and passed; against the real API
  every field would decode as zero. Tags now match the documented
  upstream schema; `Side`, `MarketID`, and `UnrealizedPnl` are
  removed (the API doesn't return them); `CashPnl`/`PercentPnl` take
  over from `UnrealizedPnl`.
- **Wrong-window market trap.** Live evidence on 2026-05-09: SOL
  08:20 UTC and ETH 07:40/08:00 signals filled buys against future
  market windows because `ResolveTokenIDsAt` silently substituted a
  different market when the slug-hit returned the wrong `startDate`.
  Fixed with the `StatusWindowMismatch` fail-closed signal and the
  strict `ResolveTokenIDsForWindow` entry point.
- **Deposit-wallet deploy false-negative trap.** The relayer `/deployed`
  endpoint can return `false` after a stale `WALLET-CREATE` row is marked
  `STATE_FAILED` even when the deposit wallet is fully deployed on Polygon.
  Polygolem and the go-bot SDK now treat `eth_getCode` at the derived
  deposit-wallet address as the source of truth.
  - `polygolem deposit-wallet status` falls back to `eth_getCode` when the
    relayer reports not deployed; the JSON envelope adds
    `relayerDeployed`, `onchainCodeDeployed`, and `deploymentStatusSource`,
    and renames the long-standing `wallerNonce` typo to `walletNonce`.
  - `polygolem deposit-wallet deploy --wait` checks `eth_getCode` before
    submitting `WALLET-CREATE` and exits with `state=already_deployed` when
    the wallet already has code. New `--rpc-url` flag overrides the
    Polygon RPC endpoint (default: `POLYGON_RPC_URL` env, then public node).
  - `pkg/relayer.DepositWalletAddress` and
    `pkg/relayer.DepositWalletCodeDeployed` (wraps `internal/rpc.HasCode`)
    expose the dual-source check to SDK consumers.
  - `go-bot/internal/polygolem.Client.DepositWalletStatus` treats on-chain
    code as the source of truth when the relayer reports false.

## [0.1.0] — 2026-05-07

First tagged release. Includes everything shipped through Phase 0–E plus
the May 2026 deposit-wallet migration and the documentation overhaul.

### Added

- **Builder auto — programmatic CLOB L2 credentials.** `polygolem builder auto`
  mints CLOB L2 HMAC credentials via local ClobAuth EIP-712 signing. Single
  env var (`POLYMARKET_PRIVATE_KEY`) required. See `docs/ONBOARDING.md`.
- **Universal market data client (`pkg/universal`).** Single client wrapping
  Gamma + CLOB + Data API + Discovery + Stream (70+ methods). Query all
  Polymarket public data through one typed surface.
- **Full Gamma API surface (`pkg/gamma`, 26 methods).** MarketBySlug,
  EventBySlug, SeriesByID, TagByID/TagBySlug, RelatedTagsByID/BySlug,
  Teams, CommentByID, CommentsByUser, PublicProfile, SportsMarketTypes,
  MarketByToken, EventsKeyset, MarketsKeyset.
- **CLOB V2 order management.** Cancel order (`clob cancel`), cancel all
  (`clob cancel-all`), typed `OrderRecord` and `TradeRecord` responses
  (replacing `json.RawMessage`), GTD expiration support
  (`--expiration` flag).
- **CreateBuilderFeeKey.** `POST /auth/builder-api-key` via L2 HMAC auth.
  Mints builder fee key for V2 order `builder` field attribution. Fully
  headless — no cookie, no browser.
- **SDK contracts documented.** All public types and method signatures in
  `pkg/` documented as Go interface contracts in Astro docs.
- **Polytypes reference.** V2 data types (`Market`, `Event`, `OrderBook`,
  `signedOrderPayload`, `EnrichedMarket`, `PriceHistory`, `OrderRecord`,
  `TradeRecord`, `CancelOrdersResponse`) documented with JSON field tags.
- **Deposit wallet pipeline documentation.** `docs/ONBOARDING.md`
  with full pipeline (derive → deploy → approve → fund → onboard),
  requirements checklist, gas sponsorship breakdown, replication steps.
  `docs/CONTRACTS.md` with all smart contract addresses, factory ABI,
  CREATE2 derivation, permission model, alternate deployment paths.
- **Astro docs site (25+ pages).** Guides (Builder Auto, Universal Client,
  Market Discovery, Deposit Wallet Lifecycle, Orderbook Data, Paper Trading,
  Bridge & Funding, Go-Bot Integration), Concepts (API Overview, Smart
  Contracts, POLY_1271 Deposit Wallets, Secrets, Markets/Events/Tokens,
  Safety, Architecture), Reference (CLI, Go SDK Contracts, Protocol Types,
  Internal Packages, Gamma/CLOB/Data/Stream APIs, Coverage Matrix).
- **Polydart PRD.** `docs/PRD_POLYDART.md` — companion Dart SDK design for
  Arenaton Flutter with Reown/WalletConnect, server proxy, confirmed
  pipeline.
- **Test coverage.** Added tests for `internal/errors`,
  `internal/marketdiscovery`, `internal/stream`. 29/29 packages pass
  CI (gofmt + vet + test).
- **Orderbook taxonomy.** `pkg/orderbook` re-exports with typed reader
  interface from `pkg/bookreader`.

### Changed

- **CLOB API reference updated for V2.** Accurate commands, POLY_1271
  signing flow, ERC-7739 TypedDataSign wrapper documentation, V2 order
  envelope fields.
- **Safety model extended for deposit wallet V2.** Signer vs funder
  separation, builder credential isolation, deposit-wallet balance routing,
  relayer auth vs trading auth rules.
- **Architecture updated.** 6 `pkg/` + 21 `internal/` packages documented
  with dependency direction diagram.
- **README rewritten.** One env var focus, accurate command inventory,
  builder auto front-and-center, SDK tables, docs links.
- **Credential documentation.** Split three credential types: CLOB L2 Trading Key
  (headless for existing users), Builder Fee Key (headless via L2 HMAC), Relayer API Key
  (headless via SIWE). See `docs/ONBOARDING.md`.

[Unreleased]: https://github.com/TrebuchetDynamics/polygolem/compare/v0.5.0...HEAD
[v0.5.0]: https://github.com/TrebuchetDynamics/polygolem/compare/v0.4.2...v0.5.0
[v0.4.2]: https://github.com/TrebuchetDynamics/polygolem/compare/v0.4.1...v0.4.2
[v0.4.1]: https://github.com/TrebuchetDynamics/polygolem/compare/v0.4.0...v0.4.1
[v0.4.0]: https://github.com/TrebuchetDynamics/polygolem/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/TrebuchetDynamics/polygolem/compare/v0.2.1...v0.3.0
[v0.2.1]: https://github.com/TrebuchetDynamics/polygolem/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/TrebuchetDynamics/polygolem/releases/tag/v0.2.0
[v0.1.1]: https://github.com/TrebuchetDynamics/polygolem/releases/tag/v0.1.1
[v2026.5.9]: https://github.com/TrebuchetDynamics/polygolem/releases/tag/v2026.5.9
[0.1.0]: https://github.com/TrebuchetDynamics/polygolem/releases/tag/v0.1.0
