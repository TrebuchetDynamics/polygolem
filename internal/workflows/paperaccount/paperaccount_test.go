package paperaccount

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/paper"
)

type fakePricer struct {
	price   string
	tokenID string
	side    string
}

func (f *fakePricer) Price(_ context.Context, tokenID, side string) (string, error) {
	f.tokenID = tokenID
	f.side = side
	return f.price, nil
}

func TestRunnerBuyUsesBestAskWhenPriceUnset(t *testing.T) {
	pricer := &fakePricer{price: "0.42"}
	runner := New(Config{State: paper.NewState("USD", 100), Pricer: pricer})

	got, err := runner.Buy(context.Background(), TradeRequest{TokenID: "token-1", Size: "2"})
	if err != nil {
		t.Fatalf("Buy returned error: %v", err)
	}
	if got.Action != "buy" || got.TokenID != "token-1" || got.Price != 0.42 || got.Size != 2 || got.Cost != 0.84 || got.Cash != 99.16 {
		t.Fatalf("response=%+v", got)
	}
	if pricer.tokenID != "token-1" || pricer.side != "SELL" {
		t.Fatalf("price call token=%q side=%q", pricer.tokenID, pricer.side)
	}
}

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

func TestRunnerRequiresTokenBeforePricing(t *testing.T) {
	pricer := &fakePricer{price: "0.42"}
	runner := New(Config{State: paper.NewState("USD", 100), Pricer: pricer})

	_, err := runner.Buy(context.Background(), TradeRequest{Size: "1"})
	if err == nil || !strings.Contains(err.Error(), "--token-id required") {
		t.Fatalf("error=%v, want --token-id required", err)
	}
	if pricer.tokenID != "" || pricer.side != "" {
		t.Fatalf("pricer called before validation token=%q side=%q", pricer.tokenID, pricer.side)
	}
}

func TestRunnerPositionsAndResetUseSameState(t *testing.T) {
	state := paper.NewState("USD", 10)
	runner := New(Config{State: state})
	if _, err := runner.Buy(context.Background(), TradeRequest{TokenID: "token-1", Price: "0.5", Size: "2"}); err != nil {
		t.Fatalf("Buy returned error: %v", err)
	}
	positions := runner.Positions()
	if positions.Cash != 9 || len(positions.Positions) != 1 || len(positions.Fills) != 1 {
		t.Fatalf("positions=%+v", positions)
	}

	reset := runner.Reset(25)
	if reset.Status != "reset" || reset.Cash != 25 {
		t.Fatalf("reset=%+v", reset)
	}
	positions = runner.Positions()
	if positions.Cash != 25 || len(positions.Positions) != 0 || len(positions.Fills) != 0 {
		t.Fatalf("positions after reset=%+v", positions)
	}
}
