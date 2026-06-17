package paper

import "testing"

func TestBuyUpdatesLocalPositionWithoutExternalExecution(t *testing.T) {
	state := NewState("USD", 100)
	fill, err := state.Buy(Order{
		MarketID: "market-1",
		TokenID:  "yes-token",
		Price:    0.25,
		Size:     10,
	})
	if err != nil {
		t.Fatalf("Buy returned error: %v", err)
	}
	if fill.Live {
		t.Fatal("paper fill must not be live")
	}
	if state.Cash != 97.5 {
		t.Fatalf("Cash = %v, want 97.5", state.Cash)
	}
	if state.Positions["yes-token"].Size != 10 {
		t.Fatalf("position size = %v", state.Positions["yes-token"].Size)
	}
}

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
	if fill.RealizedPnL < 0.9999999 || fill.RealizedPnL > 1.0000001 { // (0.30-0.20)*10
		t.Fatalf("RealizedPnL = %v, want ~1", fill.RealizedPnL)
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
	if fill.RealizedPnL < -1.5000001 || fill.RealizedPnL > -1.4999999 { // (0.10-0.40)*5
		t.Fatalf("RealizedPnL = %v, want ~-1.5", fill.RealizedPnL)
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
	cashBefore, costBefore, fillsBefore := state.Cash, state.Positions["yes"].Cost, len(state.Fills)
	if _, err := state.Sell(Order{TokenID: "yes", Price: 0.5, Size: 4}); err == nil {
		t.Fatal("oversell must error")
	}
	if _, err := state.Sell(Order{TokenID: "yes", Price: 0.5, Size: 0}); err == nil {
		t.Fatal("zero-size sell must error")
	}
	if state.Cash != cashBefore || state.Positions["yes"].Cost != costBefore || len(state.Fills) != fillsBefore {
		t.Fatal("rejected sells must not mutate state")
	}
}
