package universal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sdkclob "github.com/TrebuchetDynamics/polygolem/pkg/clob"
)

type haltedGate struct{}

func (haltedGate) CanProceed() bool { return false }

const gateTestPrivateKey = "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"

// TestConfigTradeGateBlocksCreateLimitOrder verifies the universal client
// threads Config.TradeGate into its order path: a halted gate blocks order
// submission before any network call.
func TestConfigTradeGateBlocksCreateLimitOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request to %s while trading halted", r.URL.Path)
	}))
	defer server.Close()

	c := NewClient(Config{CLOBBaseURL: server.URL, TradeGate: haltedGate{}})

	_, err := c.CreateLimitOrder(context.Background(), gateTestPrivateKey, sdkclob.CreateOrderParams{})
	if !errors.Is(err, ErrTradingHalted) {
		t.Fatalf("err = %v, want ErrTradingHalted", err)
	}
	// The re-exported sentinel must equal the underlying clob sentinel.
	if !errors.Is(err, sdkclob.ErrTradingHalted) {
		t.Fatalf("err = %v, want sdkclob.ErrTradingHalted", err)
	}
}

// TestConfigNoTradeGateUnchanged verifies a nil gate does not short-circuit:
// the request reaches the server (failing later for unrelated reasons).
func TestConfigNoTradeGateUnchanged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(Config{CLOBBaseURL: server.URL})

	_, err := c.CreateLimitOrder(context.Background(), gateTestPrivateKey, sdkclob.CreateOrderParams{})
	if errors.Is(err, ErrTradingHalted) {
		t.Fatalf("nil gate must not return ErrTradingHalted, got %v", err)
	}
}
