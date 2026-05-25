package streamuser

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/stream"
)

func TestRunnerConnectsSubscribesAndStopsAtMaxMessages(t *testing.T) {
	fake := &fakeStreamer{}
	runner := New(func(cfg StreamConfig, credentials auth.APIKey) Streamer {
		fake.cfg = cfg
		fake.credentials = credentials
		return fake
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var got []interface{}
	err := runner.Run(ctx, Request{
		MarketsRaw:  " condition-1, condition-2 ",
		URL:         "ws://example.test/user",
		MaxMessages: 2,
		Credentials: auth.APIKey{Key: "key", Secret: "secret", Passphrase: "pass"},
	}, func(v interface{}) { got = append(got, v) }, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !fake.connected || !fake.closed {
		t.Fatalf("connected=%v closed=%v", fake.connected, fake.closed)
	}
	if fake.cfg.URL != "ws://example.test/user" {
		t.Fatalf("URL=%q", fake.cfg.URL)
	}
	if fake.credentials.Key != "key" || fake.credentials.Secret != "secret" || fake.credentials.Passphrase != "pass" {
		t.Fatalf("credentials=%+v", fake.credentials)
	}
	if !reflect.DeepEqual(fake.markets, []string{"condition-1", "condition-2"}) {
		t.Fatalf("markets=%v", fake.markets)
	}
	if len(got) != 2 {
		t.Fatalf("events=%d, want 2", len(got))
	}
	if got[0].(stream.UserOrderMessage).OrderID != "ord-1" {
		t.Fatalf("first event=%+v", got[0])
	}
	if got[1].(stream.UserTradeMessage).TradeID != "trade-1" {
		t.Fatalf("second event=%+v", got[1])
	}
}

func TestRunnerRejectsIncompleteCredentials(t *testing.T) {
	err := New(func(StreamConfig, auth.APIKey) Streamer {
		t.Fatal("streamer should not be created")
		return nil
	}).Run(context.Background(), Request{Credentials: auth.APIKey{Key: "key"}}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "configured CLOB L2 credentials invalid") {
		t.Fatalf("error=%v", err)
	}
}

type fakeStreamer struct {
	cfg         StreamConfig
	credentials auth.APIKey
	handlers    Handlers
	markets     []string
	connected   bool
	closed      bool
}

func (f *fakeStreamer) SetHandlers(handlers Handlers) { f.handlers = handlers }

func (f *fakeStreamer) Connect(context.Context) error {
	f.connected = true
	return nil
}

func (f *fakeStreamer) SubscribeUser(_ context.Context, markets []string) error {
	f.markets = append([]string(nil), markets...)
	f.handlers.OnOrder(stream.UserOrderMessage{EventType: "order", OrderID: "ord-1"})
	f.handlers.OnTrade(stream.UserTradeMessage{EventType: "trade", TradeID: "trade-1"})
	return nil
}

func (f *fakeStreamer) Wait(ctx context.Context, done <-chan struct{}) error {
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *fakeStreamer) Close() { f.closed = true }
