package rfq

import (
	"errors"
	"testing"
	"time"
)

func TestValidateRequestAcceptsMarketOrToken(t *testing.T) {
	for _, req := range []Request{
		{MarketID: "market-1", Side: SideBuy, Amount: "10.5"},
		{TokenID: "token-1", Side: SideSell, Amount: "2"},
	} {
		if err := ValidateRequest(req); err != nil {
			t.Fatalf("ValidateRequest(%+v): %v", req, err)
		}
	}
}

func TestValidateRequestRejectsMissingRequiredFields(t *testing.T) {
	cases := []Request{
		{Side: SideBuy, Amount: "1"},
		{MarketID: "market-1", Amount: "1"},
		{MarketID: "market-1", Side: SideBuy},
		{MarketID: "market-1", Side: SideBuy, Amount: "0"},
		{MarketID: "market-1", Side: SideBuy, Amount: "0.0"},
		{MarketID: "market-1", Side: SideBuy, Amount: "-1"},
		{MarketID: "market-1", Side: SideBuy, Amount: "1e3"},
		{MarketID: "market-1", Side: SideBuy, Amount: "1.2.3"},
		{MarketID: "market-1", Side: "HOLD", Amount: "1"},
		{MarketID: "market-1", Side: SideSell, Amount: "1", Expiration: time.Now().Add(-time.Minute)},
	}
	for _, req := range cases {
		if err := ValidateRequest(req); err == nil {
			t.Fatalf("expected validation error for %+v", req)
		}
	}
}

func TestSubmitIsExplicitlyUnsupported(t *testing.T) {
	client := NewClient()
	_, err := client.Submit(Request{MarketID: "market-1", Side: SideBuy, Amount: "1"})
	if !errors.Is(err, ErrSubmitUnsupported) {
		t.Fatalf("err=%v want ErrSubmitUnsupported", err)
	}
}
