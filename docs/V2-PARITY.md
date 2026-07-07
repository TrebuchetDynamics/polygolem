# V2 Parity Audit

Date: 2026-05-10. Derived from polygolem source at this commit.

This audit answers a narrower question than `docs/POLYMARKET-COVERAGE-MATRIX.md`:
> What does "V2" mean in this codebase, and where does our V2 surface diverge
> from what Polymarket actually exposes?

It is the prerequisite for the next live-trading slice (authenticated user
WebSocket). The matrix tells you what is wired; this audit tells you what
specifically remains on the V2 side and in what priority.

## What "V2" means here

"V2" in polygolem refers to three concrete things, none of which is a REST
URL prefix:

1. **CLOB Exchange V2 contracts** on Polygon. Addresses are pinned in source:
   - `ctfExchangeV2     = 0xE111180000d2663C0091e4f400237545B87B996B` — `internal/relayer/approvals.go:15`
   - `negRiskExchangeV2 = 0xe2222d279d744050d28e00520010520000310F59` — `internal/relayer/approvals.go:16`
   - `negRiskAdapterV2  = 0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296` — `internal/relayer/approvals.go:17`
   - Same addresses also live as `clobExchangeAddress` / `negRiskExchangeAddress` in `internal/clob/orders.go:24-25` (commented "V2 regular" / "V2 neg-risk") and as typed config fields in `pkg/contracts/contracts.go:52-54`.
2. **V2 CLOB order payload schema** that those contracts accept. The signing
   path is in `internal/clob/orders.go` (`CreateLimitOrder`, `CreateMarketOrder`).
3. **V2 relayer client** — `NewV2(baseURL, key, chainID)` at
   `pkg/relayer/relayer.go:87` is the constructor used for V2 deposit-wallet
   relayer keys; `New(...)` is the legacy builder-config path.

What "V2" is **not**:

- It is not a REST path prefix. Polymarket's HTTP surface is mostly
  unversioned (`/auth/*`, `/markets/*`, `/events/*`, `/rewards/*`, `/orders/*`,
  `/data/*`, `/relayer/*`).
- The two `/v1/` paths in source (`/v1/heartbeats`, `/v1/leaderboard`) are the
  **only** paths Polymarket exposes for those features. No `/v2` variant
  exists. They are not deprecated and need no migration; this audit closes
  that loop so future readers don't chase a phantom V2 cleanup.

## REST path inventory

Enumerated from `grep "/v1/\|"/[a-z]" --include='*.go'` across non-test source.

| Prefix             | Status                              | Notes                                                      |
|--------------------|-------------------------------------|------------------------------------------------------------|
| `/v1/heartbeats`   | Used                                | `internal/clob/orders.go` — order placement heartbeats     |
| `/v1/leaderboard`  | Used                                | `internal/dataapi/client.go` — `data leaderboard` CLI       |
| `/auth/*`          | Used (5 endpoints)                  | API key + builder fee key family                           |
| `/markets/*`       | Used                                | Token/condition lookup                                     |
| `/events/*`        | Used                                | Event metadata                                             |
| `/orders/*`        | Used                                | `/orders/scoring`                                          |
| `/data/*`          | Used                                | Data API trades, orders                                    |
| `/comments/*`, `/series/*`, `/tags/*` | Used               | Gamma taxonomy                                             |
| `/rewards/*`       | Used (8 endpoints)                  | Rewards/rebates                                            |
| `/relayer/api/*`   | Used                                | Deposit-wallet relayer                                     |
| `/v2/*`            | **None exist**                      | Confirmed: zero non-test references                        |

## Surface-by-surface V2 parity

For each V2-touching surface, columns are SDK / CLI / Test status. ✅ wired,
⚠️ partial, ❌ gap.

| Surface                         | SDK | CLI | Test | Notes                                                                                                  |
|---------------------------------|-----|-----|------|--------------------------------------------------------------------------------------------------------|
| V2 limit order signing (BUY)    | ✅  | ✅  | ✅   | `exchange create-order`                                                                                    |
| V2 limit order signing (SELL)   | ✅  | ✅  | ✅   | Same path; side is a parameter                                                                         |
| V2 market order (BUY)           | ✅  | ✅  | ✅   | Buy market orders treat `amount` as USDC budget and discover price from asks when omitted                |
| V2 market order (SELL)          | ✅  | ✅  | ✅   | Sell market orders treat `amount` as share size and discover price from bids when omitted               |
| V2 cancel one / batch / all     | ✅  | ✅  | ✅   | `exchange cancel`, `cancel-orders`, `cancel-all`, `cancel-market`                                          |
| V2 builder fee key              | ✅  | ✅  | ✅   | Create/list/revoke wired                                                                               |
| V2 CLOB account reads           | ✅  | ✅  | ✅   | balance, orders, trades                                                                                |
| V2 relayer wallet-create        | ✅  | ✅  | ✅   | `wallet onboard`                                                                               |
| V2 relayer wallet-batch         | ✅  | ✅  | ✅   | Approval/redeem flows                                                                                  |
| V2 relayer transaction lookup   | ✅  | ✅  | ✅   | `Client.GetTransaction` / `PollTransaction`; CLI lookup via `tx transaction <id>` or `wallet status --tx-id <id>` |
| V2 collateral adapters          | ✅  | ✅  | ✅   | Approve/split/merge/redeem                                                                             |
| WS market channel               | ✅  | ✅  | ✅   | `wss://ws-subscriptions-clob.polymarket.com/ws/market` — `internal/stream/client.go:16`                |
| WS user channel (auth)          | ✅  | ✅  | ✅   | `pkg/stream.UserClient` and `polygolem stream user` handle authenticated order/trade events           |
| Bridge supported assets         | ✅  | ✅  | ✅   | `bridge assets`                                                                                        |
| Bridge create deposit address   | ✅  | ✅  | ✅   | `bridge deposit`                                                                                       |
| Bridge get quote                | ✅  | ✅  | ✅   | `bridge quote` posts source/destination token details and returns the Bridge quote payload               |
| Bridge get deposit status       | ✅  | ✅  | ✅   | `bridge status <deposit-address>` returns observed Bridge transactions                                  |

## Confirmed gaps (with evidence)

No confirmed V2 surface gaps remain in this focused matrix. Continue to treat live order execution, relayer submission, and approvals as safety-gated flows that require explicit operator intent and local tests.

## Non-gaps that may look like gaps

- `/v1/heartbeats` and `/v1/leaderboard`. Not legacy. Polymarket exposes no
  V2 replacement. Action: leave as-is; add a one-line note in
  `docs/POLYMARKET-COVERAGE-MATRIX.md` clarifying so the next reader does not
  open an issue against them.
- `data open-interest` requiring a token ID. Already tracked in the existing
  matrix; not a V2 parity issue, it's a response-shape capture issue.

## Recommended slice order

No remaining V2 parity slices are currently identified in this focused matrix.

## What this audit deliberately does not include

- Re-deriving the existing coverage matrix. See
  `docs/POLYMARKET-COVERAGE-MATRIX.md` for the full surface picture.
- A schedule. Slice order is given; sequencing is a separate decision.
- Schema-level diffs against Polymarket's typed protobufs / OpenAPI. Polymarket
  does not publish either; parity here is established by behavioral tests, not
  a spec diff.
