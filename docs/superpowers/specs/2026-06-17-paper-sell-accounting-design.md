# Paper Sell Accounting — Design

**Date:** 2026-06-17
**Status:** Approved design, pending implementation plan

## Problem

The paper-trading workflow's "sell" does not sell. `internal/workflows/paperaccount`
routes both buy and sell through `paper.State.Buy`, so a sell *debits* cash and
*grows* the position instead of crediting cash and reducing it. `internal/paper`
has no sell-side accounting at all, and a test
(`TestRunnerSellUsesBestBidAndPreservesLocalAccounting`) pins the wrong behavior.
Separately, `internal/paper/doc.go` claims paper state "lives entirely on disk"
with "persisted snapshots" and is written by an executor in `internal/execution`;
none of that is true — state is in-memory and per-process.

This adds correct sell accounting and corrects the misleading docs. Real on-disk
persistence is explicitly out of scope.

## Scope

### In scope
- A `paper.State.Sell` method with average-cost accounting, oversell rejection,
  and realized-PnL reporting.
- Wiring `paperaccount` sells through `Sell` instead of `Buy`.
- Reporting realized PnL on sell responses.
- Correcting `internal/paper/doc.go` to describe actual (in-memory, per-process)
  behavior.
- Replacing the test that pins the buggy behavior; adding sell-accounting tests.

### Out of scope (deliberate decisions)
- **On-disk persistence** — state remains in-memory/per-process; deferred to a
  separate feature. `doc.go` is corrected to say so.
- **FIFO lot accounting** — average cost only (matches the existing total-`Cost`
  field on `Position`).
- **Shorting** — selling more than held is an error, not a negative position
  (Polymarket outcome tokens cannot be sold short in this model).

## Current state (facts)

- `paper.Position{TokenID string; Size float64; Cost float64}` — `Cost` is the
  *total* cost basis, so average cost = `Cost / Size`.
- `paper.State{Currency; Cash float64; Positions map[string]Position; Fills []Fill}`.
- `paper.State.Buy(order)` debits `Cash` by `Price*Size`, increments
  `pos.Size`/`pos.Cost`, appends a `Fill`, errors on insufficient cash.
- `paper.Fill{MarketID; TokenID; Price; Size; Live bool}` — no side, no PnL.
- `paperaccount.trade(action, clobSide, req)` prices against `clobSide`, parses
  size, then unconditionally calls `state.Buy`; sets `res.Proceeds` for sells and
  `res.Cost` for buys. `Buy` uses `clobSide="SELL"`, `Sell` uses `clobSide="BUY"`
  (best-bid pricing for a sell) — this pricing-side behavior is correct and stays.

## Design

### 1. `paper.Fill` — add realized PnL

```go
type Fill struct {
	MarketID    string  `json:"market_id"`
	TokenID     string  `json:"token_id"`
	Price       float64 `json:"price"`
	Size        float64 `json:"size"`
	Live        bool    `json:"live"`
	RealizedPnL float64 `json:"realized_pnl,omitempty"`
}
```
`Buy` leaves `RealizedPnL` at 0; `omitempty` keeps existing Buy fill JSON
unchanged. Only sells populate it.

### 2. `paper.State.Sell`

```go
// Sell reduces a held position using average-cost accounting, credits cash with
// the proceeds, and reports realized PnL. Selling more than is held, or selling
// a token with no position, is rejected (no shorting).
func (s *State) Sell(order Order) (Fill, error)
```
Behavior:
- `const sizeEpsilon = 1e-9`.
- If `order.Size <= 0` → error `"sell size must be positive"`.
- `pos := s.Positions[order.TokenID]`. If `pos.Size <= 0` or
  `order.Size > pos.Size + sizeEpsilon` → error `"insufficient paper position"`.
- `avgCost := pos.Cost / pos.Size`.
- `proceeds := order.Price * order.Size`.
- `realized := (order.Price - avgCost) * order.Size`.
- `s.Cash += proceeds`.
- `pos.Size -= order.Size`; `pos.Cost -= avgCost * order.Size`.
- If `pos.Size <= sizeEpsilon`: `delete(s.Positions, order.TokenID)` (clear float
  dust). Else write `pos` back.
- Append `Fill{MarketID, TokenID, Price, Size, Live:false, RealizedPnL: realized}`
  and return it.

No state mutation occurs on any error path (validation happens before any write).

### 3. `paperaccount.trade` — branch on action

Replace the unconditional `r.state.Buy(...)` with:
```go
var fill paper.Fill
if action == "sell" {
	fill, err = r.state.Sell(paper.Order{TokenID: tokenID, Price: price, Size: size})
} else {
	fill, err = r.state.Buy(paper.Order{TokenID: tokenID, Price: price, Size: size})
}
if err != nil { return TradeResponse{}, err }
```
Response: for `action == "sell"` set `res.Proceeds = price * size` and
`res.RealizedPnL = fill.RealizedPnL`; for buys keep `res.Cost = price * size`.

Add `RealizedPnL float64 \`json:"realized_pnl,omitempty"\`` to `TradeResponse`.

### 4. `internal/paper/doc.go`

Rewrite to describe actual behavior: in-memory, per-process paper state (positions,
fills, cash) that never reaches an authenticated Polymarket endpoint; no on-disk
persistence (a future enhancement); used for offline development and edge
validation.

## Behavior contract

| Action | Cash | Position | Reported |
|---|---|---|---|
| Buy (held cash ≥ cost) | −price·size | +size, +cost | `Cost = price·size` |
| Sell (held size ≥ size) | +price·size | −size, −avgCost·size (deleted at 0) | `Proceeds`, `RealizedPnL` |
| Buy, insufficient cash | unchanged | unchanged | error |
| Sell, no/insufficient position | unchanged | unchanged | error |
| Sell, size ≤ 0 | unchanged | unchanged | error |

## Testing

`internal/paper/state_test.go` (new or extended):
- Buy then full sell: cash returns to ~start + PnL, position deleted, `RealizedPnL`
  = `(sellPrice − buyPrice) * size`.
- Buy then partial sell: position size/cost reduced proportionally, cash credited,
  realized PnL correct on the sold portion; remaining avg cost unchanged.
- Sell with no position → error, state unchanged.
- Oversell (size > held) → error, state unchanged.
- Sell size 0/negative → error.
- Loss case: sell below avg cost yields negative `RealizedPnL`.

`internal/workflows/paperaccount/paperaccount_test.go`:
- Replace `TestRunnerSellUsesBestBidAndPreservesLocalAccounting` with
  `TestRunnerSellReducesPositionAndCreditsCash`: buy first (to hold a position),
  then sell; assert cash **increases** by proceeds, position reduces, `Action ==
  "sell"`, `Proceeds`/`RealizedPnL` correct, and pricing side is `"BUY"`.
- Add `TestRunnerSellRejectsWhenNoPosition`: sell with no prior buy → error.

## Components and boundaries

- **`paper.State.Sell`** — pure in-memory accounting; depends only on `State`
  fields. Mirrors `Buy`.
- **`paper.Fill.RealizedPnL`** — additive, backward-compatible field.
- **`paperaccount.trade`** — orchestration only; chooses Buy vs Sell, maps to the
  response DTO. No accounting logic of its own.
- **`doc.go`** — documentation only.

No new dependencies; no change to `Buy`, pricing, or the CLI command surface.
