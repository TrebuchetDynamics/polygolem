package clob

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type haltedGate struct{}

func (haltedGate) CanProceed() bool { return false }

func TestConfigTradeGateBlocksCreateLimitOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request to %s while trading halted", r.URL.Path)
	}))
	defer server.Close()

	c := NewClient(Config{BaseURL: server.URL, TradeGate: haltedGate{}})

	_, err := c.CreateLimitOrder(context.Background(), "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d", CreateOrderParams{})
	if !errors.Is(err, ErrTradingHalted) {
		t.Fatalf("err = %v, want ErrTradingHalted", err)
	}
}

func TestConfigNoTradeGateUnchanged(t *testing.T) {
	// A nil TradeGate must not block: this client reaches the server (and fails
	// there for unrelated reasons), proving the gate did not short-circuit.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(Config{BaseURL: server.URL})
	_, err := c.CreateLimitOrder(context.Background(), "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d", CreateOrderParams{})
	if errors.Is(err, ErrTradingHalted) {
		t.Fatalf("nil gate must not return ErrTradingHalted, got %v", err)
	}
}
