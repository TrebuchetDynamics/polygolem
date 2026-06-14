package clob

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/transport"
)

// fakeGate is a test TradeGate with a fixed CanProceed result.
type fakeGate struct{ ok bool }

func (g fakeGate) CanProceed() bool { return g.ok }

// failingServer fails the test if any HTTP request reaches it.
func failingServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request to %s while trading halted", r.URL.Path)
	}))
}

func TestCreateLimitOrderBlockedWhenGateHalted(t *testing.T) {
	server := failingServer(t)
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc, WithTradeGate(fakeGate{ok: false}))

	_, err := client.CreateLimitOrder(context.Background(), testOrderPrivateKey, CreateOrderParams{})
	if !errors.Is(err, ErrTradingHalted) {
		t.Fatalf("err = %v, want ErrTradingHalted", err)
	}
}
