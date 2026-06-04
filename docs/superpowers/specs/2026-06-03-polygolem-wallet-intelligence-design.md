# Polygolem Wallet Intelligence Design

> **Status:** Draft from CrowdIntel competitive study — ready for grilling / implementation planning
> **Date:** 2026-06-03
> **Scope:** Read-only wallet intelligence, leaderboard scoring, market-flow summaries, and explainable alert signals for `polygolem`

---

## 1. Executive Summary

CrowdIntel demonstrates a strong prediction-market wedge: users want to know **which wallets are moving, which wallets have edge, and why a signal is reproducible** before they decide whether to research or trade a market.

`polygolem` already has the hard protocol foundation: Gamma discovery, CLOB market data, Data API positions/trades/activity/holders/leaderboards, public streams, read-only defaults, and a strict deposit-wallet safety model. The missing layer is a read-only derived-intelligence surface that turns existing primitives into wallet dossiers, leaderboards, market-flow summaries, and explainable statistical signals.

This design adds a non-custodial, non-accusatory intelligence layer. It does not place trades, sign payloads, approve tokens, custody funds, or claim that any wallet is definitively an insider.

---

## 2. Competitive Lessons From CrowdIntel

Observed public CrowdIntel surfaces:

| Surface | Lesson for polygolem |
|---|---|
| `/whales` | Wallet leaderboard should rank by volume, PnL, ROI, bets, win rate, and last activity. |
| `/leaderboard` | Raw win rate is not enough; use shrinkage-adjusted win rate so small samples do not dominate. |
| `/alerts` / `insider-radar` | A live/read-only signal feed can be valuable without being an execution surface; Polygolem V1 deliberately starts narrower with user-scoped dossier alerts. |
| `/insiders` | Use careful language: flagged, candidate, potential coordination, statistical signal. |
| `/pricing` | Free read-only discovery plus richer dossiers/follows/exports is a clear product boundary. |
| Landing claims | “Every number reproducible from Polygon” is a trust pattern: every score needs source metadata and formula version. |

Language rule for `polygolem`: use **flagged**, **signal**, **candidate**, **potential coordination**, and **requires review**. Avoid presenting “insider,” “fraud,” or “manipulation” as proven facts.

---

## 3. Goals

- Add a read-only wallet intelligence package that composes existing `pkg/data`, `pkg/gamma`, `pkg/clob`, and stream primitives.
- Provide deterministic, testable scoring functions for wallet edge and signal ranking.
- Make every derived value reproducible from raw source data, as-of timestamps, and formula versions.
- Preserve `polygolem` safety: no credentials required for public intelligence, no signing, no mutation, no live execution shortcuts.
- Expose CLI and SDK surfaces that bots, dashboards, and docs sites can consume without scraping Polymarket directly.

## 4. Non-Goals

- No trading strategy recommendations.
- No order placement, cancellation, approval, funding, bridge, or settlement flows.
- No private key, CLOB API key, or wallet-auth dependency.
- No legal conclusion that a wallet is an insider or engaged in misconduct.
- No full proprietary graph database in the first slice.
- No paid-gating implementation inside `polygolem`; product packaging belongs outside this SDK.

---

## 5. Proposed Architecture

### Public SDK: `pkg/intel`

`pkg/intel` owns pure, stable DTOs and scoring helpers:

| Type / Function | Purpose |
|---|---|
| `WalletSummary` | Wallet volume, PnL, ROI, bets, win rate, last activity, source freshness. |
| `WalletScore` | Shrinkage-adjusted win rate, sample confidence, category edge, concentration flags. |
| `WalletDossier` | Summary plus recent trades, current/closed positions, notable markets, and explanations. |
| `LeaderboardRow` | Ranked wallet row suitable for CLI/table/JSON output. |
| `Signal` | Explainable alert candidate with score, reasons, source IDs, and confidence. |
| `MarketFlow` | Holder/trade/open-interest summary for one market or token. |
| `ShrinkageWinRate(wins, bets, priorWins, priorBets)` | Deterministic ranking primitive. |
| `ROI(pnl, volume)` | Deterministic financial metric helper. |
| `ScoreWallet(input ScoreInput)` | Pure signal aggregation, no network calls. |

### Internal application service: `internal/intel`

`internal/intel` composes network clients and handles source freshness:

```text
internal/cli intel commands
        |
internal/intel service
        |
pkg/data + pkg/gamma + pkg/clob + pkg/marketresolver + pkg/stream
        |
pkg/intel DTOs and pure scoring helpers
```

Rules:

- `pkg/intel` must not import `internal/*`.
- Network fetching stays in `internal/intel` or existing public client packages.
- Scoring stays pure and table-testable.
- CLI output uses the existing JSON envelope / output conventions.
- Every response includes `as_of`, `source`, and `formula_version` fields where a value is derived.

---

## 6. CLI Surface

Initial commands:

```bash
polygolem --json intel wallet 0xabc...
polygolem --json intel leaderboard --limit 50 --sort data-api-rank
polygolem --json intel market-flow <market-or-token-id> --limit 100
polygolem --json intel alerts --user 0xabc... --min-score 70 --limit 50
```

Later commands:

```bash
polygolem intel follow 0xabc...          # local watchlist, no auth required
polygolem intel category-edge crypto     # category-scoped edge ranking
polygolem intel cluster 0xabc...         # only after funding graph evidence exists
```

Command safety:

- All `intel` commands are read-only.
- `intel` commands must run without `POLYMARKET_PRIVATE_KEY`.
- If a future command needs authenticated user data, it must live outside the default public `intel` surface or require an explicit mode gate.

---

## 7. Scoring Model V1

### Required fields

A wallet score must expose:

- `wallet`
- `score`
- `confidence`
- `formula_version`
- `as_of`
- `source_rows`
- `reasons[]`
- `raw_metrics`

### V1 signals

| Signal | Description | Notes |
|---|---|---|
| `sample_size` | Bet/trade count is large enough to interpret. | Prevents small-sample luck. |
| `shrinkage_win_rate` | Win rate pulled toward a configurable prior. | Primary ranking signal. |
| `positive_pnl` | Realized PnL is positive. | Must keep raw and normalized values. |
| `roi` | PnL divided by volume. | Guard against tiny-volume outliers. |
| `category_edge` | Wallet outperforms within a market category. | Requires category labels from Gamma. |
| `concentration` | Large exposure to one market/category. | Alert signal, not proof. |
| `late_entry` | Position opens close to known resolution/freshness boundary. | Requires timestamps; keep conservative. |
| `repeat_co_positioning` | Same wallets repeatedly enter same side/market. | V1 may emit only if Data API evidence is available. |

### Formula discipline

- Use semantic formula versions: `wallet_score_v1`, `shrinkage_win_rate_v1`.
- Changing weights or thresholds increments the formula version.
- Tests must pin example inputs and outputs.
- CLI must show enough `reasons[]` for a human to understand why a score changed.

---

## 8. Data Sources

| Source | Existing polygolem support | Use in intel |
|---|---|---|
| Data API positions | `pkg/data.CurrentPositions` | Wallet current exposure. |
| Data API closed positions | `pkg/data.ClosedPositions` | Realized results and historical accuracy. |
| Data API trades/activity | `pkg/data.Trades`, `Activity`, `MarketTrades` | Recency, market-flow, alert feed. |
| Data API holders | `pkg/data.TopHolders` | Market concentration and holder views. |
| Data API leaderboard | `pkg/data.TraderLeaderboard` | Seed rows for ranking and comparison. |
| Gamma markets/events/tags | `pkg/gamma`, `pkg/universal` | Category labels and market metadata. |
| CLOB books/prices | `pkg/clob`, `pkg/orderbook`, `pkg/marketdata` | Price context for market-flow summaries. |
| Polygon transfers | Future optional RPC/indexer slice | Funding edges / cluster investigations. |

V1 should avoid direct Polygon transfer graph work unless there is already a reliable source in `polygolem`. Funding clusters are valuable, but a weak cluster implementation would be worse than no cluster feature.

---

## 9. JSON Contract Sketch

### `polygolem --json intel wallet 0x...`

```json
{
  "wallet": "0xabc...",
  "as_of": "2026-06-03T00:00:00Z",
  "formula_version": "wallet_score_v1",
  "summary": {
    "volume": "123456.78",
    "realized_pnl": "2345.67",
    "roi": 0.019,
    "bets": 152,
    "wins": 91,
    "raw_win_rate": 0.5987,
    "shrinkage_win_rate": 0.5871,
    "last_active": "2026-06-03T00:00:00Z"
  },
  "score": {
    "value": 72,
    "confidence": "medium",
    "reasons": [
      "positive realized PnL over sufficient sample",
      "shrinkage-adjusted win rate above prior",
      "recent concentrated position requires review"
    ]
  },
  "sources": [
    {"kind": "data_api.closed_positions", "rows": 100},
    {"kind": "data_api.trades", "rows": 100}
  ]
}
```

### `polygolem --json intel alerts --user 0x...`

```json
{
  "dossier_alerts": [
    {
      "score": 81,
      "wallet": "0xabc...",
      "market": "",
      "side": "",
      "size": "",
      "price": "",
      "confidence": "medium",
      "formula_version": "wallet_score_v1",
      "as_of": "2026-06-03T00:00:00Z",
      "language": "statistical candidate signal; not a finding of misconduct",
      "reasons": ["wallet dossier score passed threshold", "positive historical PnL"]
    }
  ]
}
```

---

## 10. Implementation Plan

### Slice 1 — Pure scoring core

Files likely involved:

- `pkg/intel/score.go`
- `pkg/intel/types.go`
- `pkg/intel/score_test.go`

Acceptance:

- `ShrinkageWinRate` has table tests for small sample, large sample, zero bets, perfect raw record, and losing record.
- `ROI` handles zero volume safely.
- `ScoreWallet` emits deterministic score/reasons for fixture inputs.
- No network calls in `pkg/intel`.

Validation:

```bash
go test ./pkg/intel/...
```

### Slice 2 — Data-backed wallet dossier service

Files likely involved:

- `internal/intel/service.go`
- `internal/intel/service_test.go`
- `pkg/intel/types.go`

Acceptance:

- Service composes `pkg/data` client results into `WalletDossier`.
- Source rows and timestamps are preserved.
- Missing upstream data returns structured `unavailable` / `partial` classifications, not misleading zeroes.
- Uses `httptest` fixtures, not live network calls.

Validation:

```bash
go test ./internal/intel/... ./pkg/intel/...
```

### Slice 3 — CLI `intel wallet` and `intel leaderboard`

Files likely involved:

- `internal/cli/intel.go`
- `internal/output` integration if needed
- `docs/COMMANDS.md` regenerated through `go run ./cmd/polygolem_docs`

Acceptance:

- Commands are registered and appear in generated docs.
- JSON output follows the existing global `--json` envelope convention; V1 does not add per-command `--output` flags.
- No credentials are required.
- CLI tests prove no live/trading mode is entered.

Validation:

```bash
go test ./internal/cli/... ./internal/workflows/...
go run ./cmd/polygolem_docs
```

### Slice 4 — Market-flow and alerts preview

Files likely involved:

- `internal/intel/market_flow.go`
- `internal/intel/alerts.go`
- `pkg/intel/market_flow.go`

Acceptance:

- `market-flow` summarizes holders/trades/open interest with source freshness.
- `alerts` emits explainable candidate signals from recent trades/activity.
- All alert copy uses “potential” / “candidate” language.
- Thresholds and formula versions are visible in output.

Validation:

```bash
go test ./internal/intel/... ./pkg/intel/...
```

---

## 11. Safety And Compliance Requirements

- Default command mode remains read-only.
- No `intel` command accepts a private key or CLOB secret.
- No `intel` command sends transactions, signs orders, approves tokens, bridges funds, or mutates local paper state.
- Scores must be presented as research signals, not trading advice.
- Scores must be presented as statistical candidates, not misconduct findings.
- Every output that uses suspicious/flagged language must include a disclaimer field or human-readable copy equivalent to: “This is a statistical signal, not a finding of misconduct.”

---

## 12. Success Criteria

| Criterion | Target |
|---|---|
| Pure scoring tests | Deterministic table tests pass with pinned outputs. |
| CLI safety | `intel *` commands run without credentials and cannot enter live mode. |
| Reproducibility | Every score includes formula version, source rows, and as-of timestamp. |
| UX utility | Users can inspect a wallet, rank Data-API leaderboard rows, and summarize market flow from one binary. |
| Legal language | No command claims “insider” as fact; outputs use candidate/signal wording. |
| SDK boundary | Consumers can import `pkg/intel` without importing `internal/*`. |

---

## 13. Resolved Decisions From Grilling

1. **Public package name:** use stable `pkg/intel` for read-only DTOs and pure primitives. Scoring changes evolve by formula version instead of package moves. V1 Dossier Alerts reuse `wallet_score_v1`; a separate alert formula waits until alerts have independent scoring semantics.
2. **Dossier source authority:** use Polygolem-owned source adapters. In V1, Data API closed-position rows are authoritative for realized PnL and win/loss counts; trade/activity reconstruction disagreements produce partial or conflicted dossiers.
3. **Alert mode:** V1 alerts are user-scoped dossier alerts: batch/pull signals emitted when a wallet dossier score passes `--min-score`. Stream-backed and global recent-trade feed alerts are deferred until batch semantics and source authority are proven.
4. **Shrinkage prior:** `shrinkage_win_rate_v1` uses a neutral source-free default prior of 10 wins over 20 bets when callers do not supply a prior. Outputs expose the actual prior values in raw metrics.
5. **Clusters:** V1 does not emit funding-cluster or common-funder edges. It may emit conservative co-positioning candidate signals from Data API trades/activity; funding clusters wait for a dedicated reproducible Polygon/indexer source adapter.
6. **ADR:** captured in [ADR 0001 — Wallet intelligence V1 boundary](../../adr/0001-wallet-intelligence-v1-boundary.md) now that `pkg/intel` and `dossier_alerts` are public SDK/CLI contracts.

---

## 14. Recommended Next Action

Wallet-intelligence V1 slices 1–4 are implemented. Before shipping, keep docs and command copy explicit that `intel alerts` emits user-scoped dossier alerts, not a global recent-trade feed.
