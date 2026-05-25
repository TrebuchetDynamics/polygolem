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

func TestRunnerSellUsesBestBidAndPreservesLocalAccounting(t *testing.T) {
	pricer := &fakePricer{price: "0.25"}
	runner := New(Config{State: paper.NewState("USD", 10), Pricer: pricer})

	got, err := runner.Sell(context.Background(), TradeRequest{TokenID: "token-2", Size: "4"})
	if err != nil {
		t.Fatalf("Sell returned error: %v", err)
	}
	if got.Action != "sell" || got.Proceeds != 1 || got.Cash != 9 {
		t.Fatalf("response=%+v", got)
	}
	if pricer.tokenID != "token-2" || pricer.side != "BUY" {
		t.Fatalf("price call token=%q side=%q", pricer.tokenID, pricer.side)
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
