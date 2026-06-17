# Paper Sell Accounting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make paper "sell" actually sell — credit cash, reduce the held position with average-cost accounting, report realized PnL, and reject overselling — and correct the misleading persistence docs.

**Architecture:** Add `paper.State.Sell` mirroring `Buy` (in-memory average-cost math on the existing `Position{Size, Cost}` fields), add a `RealizedPnL` field to `paper.Fill`, branch `paperaccount.trade` to call `Sell` for sells, and fix `internal/paper/doc.go`. No persistence, no FIFO, no shorting.

**Tech Stack:** Go, standard `testing`.

**Spec:** `docs/superpowers/specs/2026-06-17-paper-sell-accounting-design.md`

**Git note:** This repo is on `main`; create a feature branch before the first commit and do not push to `main`. There is also a pre-existing, unrelated working-tree change to `CONTEXT.md` (the owner's "Cross-SDK Parity" glossary edit) — do NOT stage, commit, or revert it; keep it out of every commit.

## File Structure

- Modify: `internal/paper/state.go` — add `Fill.RealizedPnL` field and `State.Sell` method.
- Modify: `internal/paper/state_test.go` — add sell-accounting unit tests.
- Modify: `internal/workflows/paperaccount/paperaccount.go` — branch `trade` on action; add `TradeResponse.RealizedPnL`.
- Modify: `internal/workflows/paperaccount/paperaccount_test.go` — replace the test that pins the buggy behavior; add an oversell-rejection test.
- Modify: `internal/paper/doc.go` — correct the persistence description.

---

### Task 1: `paper.State.Sell` with average-cost accounting

**Files:**
- Modify: `internal/paper/state.go`
- Modify: `internal/paper/state_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/paper/state_test.go`:

```go
func TestSellFullPositionCreditsCashAndDeletesPosition(t *testing.T) {
	state := NewState("USD", 100)
	if _, err := state.Buy(Order{TokenID: "yes", Price: 0.20, Size: 10}); err != nil {
		t.Fatalf("Buy: %v", err)
	}
	// Cash now 98; position 10 @ avg 0.20.
	fill, err := state.Sell(Order{TokenID: "yes", Price: 0.30, Size: 10})
	if err != nil {
		t.Fatalf("Sell: %v", err)
	}
	if state.Cash != 101 { // 98 + 0.30*10
		t.Fatalf("Cash = %v, want 101", state.Cash)
	}
	if _, ok := state.Positions["yes"]; ok {
		t.Fatal("fully-sold position must be deleted")
	}
	if fill.RealizedPnL != 1 { // (0.30-0.20)*10
		t.Fatalf("RealizedPnL = %v, want 1", fill.RealizedPnL)
	}
	if fill.Live {
		t.Fatal("paper fill must not be live")
	}
}

func TestSellPartialPositionReducesSizeAndCost(t *testing.T) {
	state := NewState("USD", 100)
	if _, err := state.Buy(Order{TokenID: "yes", Price: 0.20, Size: 10}); err != nil {
		t.Fatalf("Buy: %v", err)
	}
	fill, err := state.Sell(Order{TokenID: "yes", Price: 0.50, Size: 4})
	if err != nil {
		t.Fatalf("Sell: %v", err)
	}
	pos := state.Positions["yes"]
	if pos.Size != 6 {
		t.Fatalf("remaining size = %v, want 6", pos.Size)
	}
	// Avg cost 0.20 unchanged: remaining cost = 6 * 0.20 = 1.2.
	if pos.Cost < 1.1999999 || pos.Cost > 1.2000001 {
		t.Fatalf("remaining cost = %v, want ~1.2", pos.Cost)
	}
	if fill.RealizedPnL < 1.1999999 || fill.RealizedPnL > 1.2000001 { // (0.50-0.20)*4
		t.Fatalf("RealizedPnL = %v, want ~1.2", fill.RealizedPnL)
	}
}

func TestSellBelowCostYieldsNegativePnL(t *testing.T) {
	state := NewState("USD", 100)
	if _, err := state.Buy(Order{TokenID: "yes", Price: 0.40, Size: 5}); err != nil {
		t.Fatalf("Buy: %v", err)
	}
	fill, err := state.Sell(Order{TokenID: "yes", Price: 0.10, Size: 5})
	if err != nil {
		t.Fatalf("Sell: %v", err)
	}
	if fill.RealizedPnL != -1.5 { // (0.10-0.40)*5
		t.Fatalf("RealizedPnL = %v, want -1.5", fill.RealizedPnL)
	}
}

func TestSellRejectsOversellAndMissingPositionAndBadSize(t *testing.T) {
	state := NewState("USD", 100)
	if _, err := state.Sell(Order{TokenID: "none", Price: 0.5, Size: 1}); err == nil {
		t.Fatal("sell with no position must error")
	}
	if _, err := state.Buy(Order{TokenID: "yes", Price: 0.20, Size: 3}); err != nil {
		t.Fatalf("Buy: %v", err)
	}
	cashBefore, costBefore := state.Cash, state.Positions["yes"].Cost
	if _, err := state.Sell(Order{TokenID: "yes", Price: 0.5, Size: 4}); err == nil {
		t.Fatal("oversell must error")
	}
	if _, err := state.Sell(Order{TokenID: "yes", Price: 0.5, Size: 0}); err == nil {
		t.Fatal("zero-size sell must error")
	}
	if state.Cash != cashBefore || state.Positions["yes"].Cost != costBefore {
		t.Fatal("rejected sells must not mutate state")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/paper/ -run TestSell`
Expected: FAIL — compile error `state.Sell undefined` and `fill.RealizedPnL undefined`.

- [ ] **Step 3: Add the `RealizedPnL` field to `Fill`**

In `internal/paper/state.go`, change the `Fill` struct:

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

- [ ] **Step 4: Implement `State.Sell`**

In `internal/paper/state.go`, add after the `Buy` method:

```go
const sizeEpsilon = 1e-9

// Sell reduces a held position using average-cost accounting, credits cash with
// the proceeds, and reports realized PnL on the returned fill. Selling more than
// is held, selling a token with no position, or a non-positive size is rejected
// (no shorting). State is not mutated on any error path.
func (s *State) Sell(order Order) (Fill, error) {
	if order.Size <= 0 {
		return Fill{}, fmt.Errorf("sell size must be positive")
	}
	pos := s.Positions[order.TokenID]
	if pos.Size <= 0 || order.Size > pos.Size+sizeEpsilon {
		return Fill{}, fmt.Errorf("insufficient paper position")
	}
	avgCost := pos.Cost / pos.Size
	proceeds := order.Price * order.Size
	realized := (order.Price - avgCost) * order.Size

	s.Cash += proceeds
	pos.Size -= order.Size
	pos.Cost -= avgCost * order.Size
	if pos.Size <= sizeEpsilon {
		delete(s.Positions, order.TokenID)
	} else {
		s.Positions[order.TokenID] = pos
	}

	fill := Fill{MarketID: order.MarketID, TokenID: order.TokenID, Price: order.Price, Size: order.Size, Live: false, RealizedPnL: realized}
	s.Fills = append(s.Fills, fill)
	return fill, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/paper/ -v -run 'TestSell|TestBuy'`
Expected: PASS for all sell tests and the existing buy test.

- [ ] **Step 6: Commit**

```bash
git add internal/paper/state.go internal/paper/state_test.go
git commit -m "feat(paper): add State.Sell with average-cost accounting and realized PnL"
```
Append this trailer to the commit body:
```
Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
```

---

### Task 2: Route paperaccount sells through `Sell`

**Files:**
- Modify: `internal/workflows/paperaccount/paperaccount.go`
- Modify: `internal/workflows/paperaccount/paperaccount_test.go`

- [ ] **Step 1: Replace the test that pins the buggy behavior**

In `internal/workflows/paperaccount/paperaccount_test.go`, DELETE `TestRunnerSellUsesBestBidAndPreservesLocalAccounting` (the one asserting a sell drops cash to 9) and add:

```go
func TestRunnerSellReducesPositionAndCreditsCash(t *testing.T) {
	pricer := &fakePricer{price: "0.30"}
	state := paper.NewState("USD", 10)
	runner := New(Config{State: state, Pricer: pricer})

	// Buy 4 @ explicit 0.20 first (explicit price skips the pricer): cash 10 -> 9.2.
	if _, err := runner.Buy(context.Background(), TradeRequest{TokenID: "token-2", Price: "0.20", Size: "4"}); err != nil {
		t.Fatalf("Buy returned error: %v", err)
	}
	// Sell 4 with no explicit price -> pricer best-bid 0.30 on side BUY: cash 9.2 -> 10.4.
	got, err := runner.Sell(context.Background(), TradeRequest{TokenID: "token-2", Size: "4"})
	if err != nil {
		t.Fatalf("Sell returned error: %v", err)
	}
	if got.Action != "sell" || got.Price != 0.30 || got.Size != 4 {
		t.Fatalf("response=%+v", got)
	}
	if got.Proceeds != 1.2 || got.Cash < 10.3999999 || got.Cash > 10.4000001 {
		t.Fatalf("proceeds/cash wrong: %+v", got)
	}
	if got.RealizedPnL < 0.3999999 || got.RealizedPnL > 0.4000001 { // (0.30-0.20)*4
		t.Fatalf("RealizedPnL = %v, want ~0.4", got.RealizedPnL)
	}
	if _, ok := state.Positions["token-2"]; ok {
		t.Fatal("fully-sold position must be deleted")
	}
	if pricer.tokenID != "token-2" || pricer.side != "BUY" {
		t.Fatalf("price call token=%q side=%q", pricer.tokenID, pricer.side)
	}
}

func TestRunnerSellRejectsWhenNoPosition(t *testing.T) {
	pricer := &fakePricer{price: "0.25"}
	runner := New(Config{State: paper.NewState("USD", 10), Pricer: pricer})

	_, err := runner.Sell(context.Background(), TradeRequest{TokenID: "token-2", Size: "4"})
	if err == nil {
		t.Fatal("sell with no position must return an error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/workflows/paperaccount/ -run 'TestRunnerSell'`
Expected: FAIL — `TestRunnerSellReducesPositionAndCreditsCash` fails because the sell currently routes through `Buy` (cash goes down, `RealizedPnL` is 0 / field missing), and `TestRunnerSellRejectsWhenNoPosition` fails because the buggy Buy-path succeeds instead of erroring.

- [ ] **Step 3: Add `RealizedPnL` to `TradeResponse`**

In `internal/workflows/paperaccount/paperaccount.go`, change the `TradeResponse` struct to add the field after `Proceeds`:

```go
type TradeResponse struct {
	Action      string     `json:"action"`
	TokenID     string     `json:"token_id"`
	Price       float64    `json:"price"`
	Size        float64    `json:"size"`
	Cost        float64    `json:"cost,omitempty"`
	Proceeds    float64    `json:"proceeds,omitempty"`
	RealizedPnL float64    `json:"realized_pnl,omitempty"`
	Cash        float64    `json:"cash"`
	Fill        paper.Fill `json:"fill"`
}
```

- [ ] **Step 4: Branch `trade` on action**

In `internal/workflows/paperaccount/paperaccount.go`, replace this block in `trade`:

```go
	fill, err := r.state.Buy(paper.Order{TokenID: tokenID, Price: price, Size: size})
	if err != nil {
		return TradeResponse{}, err
	}
	res := TradeResponse{Action: action, TokenID: tokenID, Price: price, Size: size, Cash: r.state.Cash, Fill: fill}
	if action == "sell" {
		res.Proceeds = price * size
	} else {
		res.Cost = price * size
	}
	return res, nil
```

with:

```go
	order := paper.Order{TokenID: tokenID, Price: price, Size: size}
	var fill paper.Fill
	if action == "sell" {
		fill, err = r.state.Sell(order)
	} else {
		fill, err = r.state.Buy(order)
	}
	if err != nil {
		return TradeResponse{}, err
	}
	res := TradeResponse{Action: action, TokenID: tokenID, Price: price, Size: size, Cash: r.state.Cash, Fill: fill}
	if action == "sell" {
		res.Proceeds = price * size
		res.RealizedPnL = fill.RealizedPnL
	} else {
		res.Cost = price * size
	}
	return res, nil
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/workflows/paperaccount/ -v -run 'TestRunner'`
Expected: PASS for all runner tests (buy, the new sell, oversell rejection, token-required, positions/reset).

- [ ] **Step 6: Commit**

```bash
git add internal/workflows/paperaccount/paperaccount.go internal/workflows/paperaccount/paperaccount_test.go
git commit -m "fix(paper): route paperaccount sells through State.Sell and report realized PnL"
```
Append the trailer:
```
Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
```

---

### Task 3: Correct the persistence docs

**Files:**
- Modify: `internal/paper/doc.go`

- [ ] **Step 1: Rewrite `doc.go`**

Replace the entire contents of `internal/paper/doc.go` with:

```go
// Package paper holds local-only paper-trading state — cash, positions, and
// fills — used to simulate buys and sells without touching any authenticated
// Polymarket endpoint.
//
// State is in-memory and per-process: each CLI invocation starts from a fresh
// account, and nothing is written to disk. (On-disk persistence across runs is
// a possible future enhancement and is intentionally not implemented yet.)
// Useful for offline development and edge validation.
//
// This package is internal and not part of the polygolem public SDK.
package paper
```

- [ ] **Step 2: Verify the package still builds and documents cleanly**

Run: `go build ./internal/paper/ && go vet ./internal/paper/`
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/paper/doc.go
git commit -m "docs(paper): describe actual in-memory per-process state, not on-disk persistence"
```
Append the trailer:
```
Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>
```

---

### Task 4: Full verification

- [ ] **Step 1: Vet and full test suite**

Run: `go vet ./... && go test -short ./...`
Expected: vet clean; all tests pass (the previously-buggy sell test is gone, replaced by the corrected ones).

- [ ] **Step 2: Confirm no stray references to old behavior**

Run: `git grep -n "PreservesLocalAccounting"`
Expected: no matches (the buggy test was removed).

---

## Self-Review

**Spec coverage:**
- `State.Sell` avg-cost / reject-oversell / realized PnL → Task 1 (method + 4 unit tests).
- `Fill.RealizedPnL` additive field → Task 1 Step 3.
- Full-sell deletes position; partial reduces size+cost → Task 1 tests + impl (`delete` on `<= sizeEpsilon`).
- Reject size ≤ 0, no position, oversell; no mutation on error → Task 1 `TestSellRejectsOversellAndMissingPositionAndBadSize`.
- `paperaccount.trade` branches Buy/Sell; reports `Proceeds` + `RealizedPnL` → Task 2.
- `TradeResponse.RealizedPnL` → Task 2 Step 3.
- Replace the pinning test; add oversell-rejection runner test; keep best-bid `"BUY"` side assertion → Task 2 Step 1.
- `doc.go` corrected → Task 3.
- Out-of-scope (persistence, FIFO, shorting) → not implemented, by design.

**Placeholder scan:** none — every step has concrete code/commands and expected output.

**Type consistency:** `State.Sell(order Order) (Fill, error)`, `Fill.RealizedPnL`, `TradeResponse.RealizedPnL`, `sizeEpsilon`, and the `Order{TokenID, Price, Size}` literal are used identically across tasks and match the real source (`Position{Size, Cost}`, `State{Cash, Positions, Fills}`, `trade(action, clobSide, req)`).
