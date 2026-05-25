package streammarket

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/stream"
)

type fakeStreamer struct {
	config       StreamConfig
	handlers     Handlers
	connected    bool
	closed       bool
	subscribedTo []string
	connectErr   error
	subscribeErr error
	waitErr      error
	onSubscribe  func(*fakeStreamer)
}

func (f *fakeStreamer) SetHandlers(handlers Handlers) { f.handlers = handlers }
func (f *fakeStreamer) Connect(ctx context.Context) error {
	f.connected = true
	return f.connectErr
}
func (f *fakeStreamer) SubscribeAssets(ctx context.Context, assetIDs []string) error {
	f.subscribedTo = append([]string(nil), assetIDs...)
	if f.onSubscribe != nil {
		f.onSubscribe(f)
	}
	return f.subscribeErr
}
func (f *fakeStreamer) Wait(ctx context.Context, done <-chan struct{}) error {
	select {
	case <-done:
		return nil
	default:
	}
	return f.waitErr
}
func (f *fakeStreamer) Close() { f.closed = true }

func TestRunnerParsesAssetsConfiguresStreamAndStopsAfterMaxMessages(t *testing.T) {
	fake := &fakeStreamer{onSubscribe: func(s *fakeStreamer) {
		s.handlers.OnBook(stream.BookMessage{EventType: "book", AssetID: "tok-a"})
		s.handlers.OnLastTrade(stream.LastTradeMessage{EventType: "last_trade_price", AssetID: "tok-b"})
	}}
	var emitted []interface{}

	err := New(func(config StreamConfig) Streamer {
		fake.config = config
		return fake
	}).Run(context.Background(), Request{
		AssetIDsRaw:    " tok-a, tok-b ",
		URL:            "wss://example.test/ws",
		MaxMessages:    1,
		CustomFeatures: true,
		Level:          2,
	}, func(v interface{}) { emitted = append(emitted, v) }, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if fake.config.URL != "wss://example.test/ws" || !fake.config.CustomFeatures || fake.config.Level != 2 {
		t.Fatalf("unexpected config: %+v", fake.config)
	}
	if !fake.connected || !fake.closed {
		t.Fatalf("connected=%v closed=%v, want both true", fake.connected, fake.closed)
	}
	if want := []string{"tok-a", "tok-b"}; !reflect.DeepEqual(fake.subscribedTo, want) {
		t.Fatalf("subscribedTo=%v, want %v", fake.subscribedTo, want)
	}
	if len(emitted) != 1 {
		t.Fatalf("emitted=%d, want 1: %+v", len(emitted), emitted)
	}
	book, ok := emitted[0].(stream.BookMessage)
	if !ok || book.AssetID != "tok-a" {
		t.Fatalf("unexpected first event: %#v", emitted[0])
	}
}

func TestRunnerUsesExplicitAssetIDs(t *testing.T) {
	fake := &fakeStreamer{onSubscribe: func(s *fakeStreamer) {
		s.handlers.OnBestBidAsk(stream.BestBidAskMessage{EventType: "best_bid_ask", AssetID: "explicit"})
	}}
	var emitted []interface{}

	err := New(func(config StreamConfig) Streamer { return fake }).Run(context.Background(), Request{
		AssetIDs:    []string{"explicit"},
		AssetIDsRaw: "ignored",
		MaxMessages: 1,
	}, func(v interface{}) { emitted = append(emitted, v) }, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if want := []string{"explicit"}; !reflect.DeepEqual(fake.subscribedTo, want) {
		t.Fatalf("subscribedTo=%v, want %v", fake.subscribedTo, want)
	}
	if len(emitted) != 1 {
		t.Fatalf("emitted=%d, want 1", len(emitted))
	}
}

func TestRunnerRequiresAssetIDsBeforeCreatingStreamer(t *testing.T) {
	var newStreamerCalled bool
	err := New(func(config StreamConfig) Streamer {
		newStreamerCalled = true
		return &fakeStreamer{}
	}).Run(context.Background(), Request{AssetIDsRaw: " , "}, func(v interface{}) {}, nil)
	if err == nil || err.Error() != "--asset-ids required" {
		t.Fatalf("error=%v, want --asset-ids required", err)
	}
	if newStreamerCalled {
		t.Fatal("streamer created despite missing asset IDs")
	}
}

func TestRunnerPropagatesConnectSubscribeAndWaitErrors(t *testing.T) {
	for name, fake := range map[string]*fakeStreamer{
		"connect":   {connectErr: errors.New("connect failed")},
		"subscribe": {subscribeErr: errors.New("subscribe failed")},
		"wait":      {waitErr: errors.New("wait failed")},
	} {
		t.Run(name, func(t *testing.T) {
			err := New(func(config StreamConfig) Streamer { return fake }).Run(context.Background(), Request{AssetIDs: []string{"tok"}}, func(v interface{}) {}, nil)
			want := fake.connectErr
			if want == nil {
				want = fake.subscribeErr
			}
			if want == nil {
				want = fake.waitErr
			}
			if !errors.Is(err, want) {
				t.Fatalf("error=%v, want %v", err, want)
			}
		})
	}
}

func TestRunnerReportsStreamErrorsOnlyForUnboundedStreams(t *testing.T) {
	streamErr := errors.New("read failed")
	fake := &fakeStreamer{onSubscribe: func(s *fakeStreamer) {
		s.handlers.OnError(streamErr)
	}}
	var reported []error

	err := New(func(config StreamConfig) Streamer { return fake }).Run(context.Background(), Request{AssetIDs: []string{"tok"}, MaxMessages: 0}, func(v interface{}) {}, func(err error) {
		reported = append(reported, err)
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(reported) != 1 || !errors.Is(reported[0], streamErr) {
		t.Fatalf("reported=%v, want %v", reported, streamErr)
	}

	reported = nil
	fake = &fakeStreamer{onSubscribe: func(s *fakeStreamer) {
		s.handlers.OnError(streamErr)
		s.handlers.OnBook(stream.BookMessage{EventType: "book", AssetID: "tok"})
	}}
	err = New(func(config StreamConfig) Streamer { return fake }).Run(context.Background(), Request{AssetIDs: []string{"tok"}, MaxMessages: 1}, func(v interface{}) {}, func(err error) {
		reported = append(reported, err)
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(reported) != 0 {
		t.Fatalf("reported=%v, want none for bounded stream", reported)
	}
}
