package clob

import (
	"math/big"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
)

func TestLimitOrderTypesAccepted(t *testing.T) {
	for _, want := range []string{"GTC", "GTD", "FOK", "FAK"} {
		got, err := parseLimitOrderType(want)
		if err != nil || got != want {
			t.Fatalf("parseLimitOrderType(%q)=(%q,%v), want %q,nil", want, got, err, want)
		}
	}
}

func TestMarketOrderTypesAccepted(t *testing.T) {
	for _, want := range []string{"FOK", "FAK"} {
		got, err := parseMarketOrderType(want)
		if err != nil || got != want {
			t.Fatalf("parseMarketOrderType(%q)=(%q,%v), want %q,nil", want, got, err, want)
		}
	}
}

func TestGTD_ExpirationPassesThrough(t *testing.T) {
	signer, err := auth.NewPrivateKeySigner(testOrderPrivateKey, polygonChainID)
	if err != nil {
		t.Fatal(err)
	}
	expirationUnix := "1778125000123"
	tokenID := big.NewInt(12345)
	order, err := buildSignedOrderPayload(signer, orderDraft{
		tokenID:     tokenID,
		side:        "BUY",
		makerAmount: "700000",
		takerAmount: "1400000",
		orderType:   "GTD",
		expiration:  expirationUnix,
	}, time.UnixMilli(1778125000123), false)
	if err != nil {
		t.Fatal(err)
	}
	if order.Expiration != expirationUnix {
		t.Fatalf("expiration=%q want %q", order.Expiration, expirationUnix)
	}
}

func TestGTC_DefaultExpirationZero(t *testing.T) {
	signer, err := auth.NewPrivateKeySigner(testOrderPrivateKey, polygonChainID)
	if err != nil {
		t.Fatal(err)
	}
	tokenID := big.NewInt(12345)
	order, err := buildSignedOrderPayload(signer, orderDraft{
		tokenID:     tokenID,
		side:        "BUY",
		makerAmount: "700000",
		takerAmount: "1400000",
		orderType:   "GTC",
		expiration:  "0",
	}, time.UnixMilli(1778125000123), false)
	if err != nil {
		t.Fatal(err)
	}
	if order.Expiration != "0" {
		t.Fatalf("GTC expiration=%q want 0", order.Expiration)
	}
}

func TestInvalidOrderTypeRejected(t *testing.T) {
	if _, err := parseLimitOrderType("INVALID"); err == nil {
		t.Fatal("parseLimitOrderType accepted invalid type")
	}
	if _, err := parseMarketOrderType("GTC"); err == nil {
		t.Fatal("parseMarketOrderType accepted resting type")
	}
}

func TestEmptyOrderTypeUsesDefaults(t *testing.T) {
	if got, err := parseLimitOrderType(""); err != nil || got != "GTC" {
		t.Fatalf("parseLimitOrderType empty=(%q,%v), want GTC,nil", got, err)
	}
	if got, err := parseMarketOrderType(""); err != nil || got != "FOK" {
		t.Fatalf("parseMarketOrderType empty=(%q,%v), want FOK,nil", got, err)
	}
}

func TestCreateOrderParams_HasExpirationField(t *testing.T) {
	p := CreateOrderParams{
		TokenID:    "123",
		Side:       "BUY",
		Price:      "0.5",
		Size:       "10",
		OrderType:  "GTD",
		Expiration: "1778125000123",
	}
	if p.Expiration != "1778125000123" {
		t.Fatalf("Expiration field not set: %s", p.Expiration)
	}
}
