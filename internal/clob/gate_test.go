package clob

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/risk"
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

func TestCreateMarketOrderBlockedWhenGateHalted(t *testing.T) {
	server := failingServer(t)
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc, WithTradeGate(fakeGate{ok: false}))

	_, err := client.CreateMarketOrder(context.Background(), testOrderPrivateKey, MarketOrderParams{})
	if !errors.Is(err, ErrTradingHalted) {
		t.Fatalf("err = %v, want ErrTradingHalted", err)
	}
}

func TestCreateBatchOrdersBlockedWhenGateHalted(t *testing.T) {
	server := failingServer(t)
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc, WithTradeGate(fakeGate{ok: false}))

	_, err := client.CreateBatchOrders(context.Background(), testOrderPrivateKey, []CreateOrderParams{{}})
	if !errors.Is(err, ErrTradingHalted) {
		t.Fatalf("err = %v, want ErrTradingHalted", err)
	}
}

// A *risk.Breaker must be usable directly as a TradeGate. This is the intended
// production wiring: a bot constructs a Breaker and passes it as the gate.
func TestRiskBreakerSatisfiesTradeGate(t *testing.T) {
	var _ TradeGate = risk.NewBreaker(risk.DefaultPolicy())

	server := failingServer(t)
	defer server.Close()

	breaker := risk.NewBreaker(risk.DefaultPolicy())
	breaker.Halt() // bot decides to stop trading

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc, WithTradeGate(breaker))

	_, err := client.CreateLimitOrder(context.Background(), testOrderPrivateKey, CreateOrderParams{})
	if !errors.Is(err, ErrTradingHalted) {
		t.Fatalf("err = %v, want ErrTradingHalted", err)
	}
}

// Cancellation must always be allowed so a halted bot can reduce exposure.
func TestCancelOrdersNotBlockedWhenGateHalted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/derive-api-key":
			_, _ = w.Write([]byte(`{"apiKey":"owner-key","secret":"c2VjcmV0","passphrase":"pass"}`))
		case "/orders":
			_, _ = w.Write([]byte(`{"canceled":["0x1"],"not_canceled":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc, WithTradeGate(fakeGate{ok: false}))

	res, err := client.CancelOrders(context.Background(), testOrderPrivateKey, []string{"0x1"})
	if err != nil {
		t.Fatalf("cancel returned error while halted: %v", err)
	}
	if errors.Is(err, ErrTradingHalted) {
		t.Fatal("cancel must not be blocked by the trade gate")
	}
	if len(res.Canceled) != 1 {
		t.Fatalf("res = %+v", res)
	}
}
