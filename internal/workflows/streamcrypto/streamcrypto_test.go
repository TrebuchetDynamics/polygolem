package streamcrypto

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/stream"
)

type fakeSearcher struct {
	params *polytypes.SearchParams
	resp   *polytypes.SearchResponse
	err    error
}

func (f *fakeSearcher) Search(ctx context.Context, params *polytypes.SearchParams) (*polytypes.SearchResponse, error) {
	f.params = params
	return f.resp, f.err
}

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
func (f *fakeStreamer) Stats() stream.StreamStatsSnapshot {
	return stream.StreamStatsSnapshot{Type: "stream_stats", Stream: "market", State: "connected", AssetIDs: append([]string(nil), f.subscribedTo...)}
}

func TestRunnerDiscoversTokensEmitsStatusAndStreamsUntilMaxMessages(t *testing.T) {
	searcher := &fakeSearcher{resp: &polytypes.SearchResponse{Events: []polytypes.Event{
		{
			ID:     "event-btc",
			Title:  "BTC 5m event",
			Active: true,
			Markets: []polytypes.Market{
				{
					ID:           "market-btc",
					Question:     "BTC up or down in 5m?",
					Active:       true,
					ClobTokenIDs: `["btc-up","btc-down"]`,
				},
				{
					ID:           "market-closed",
					Question:     "BTC stale 5m?",
					Active:       true,
					Closed:       true,
					ClobTokenIDs: `["closed"]`,
				},
			},
		},
		{
			ID:     "event-eth",
			Title:  "ETH 5m event",
			Active: true,
			Markets: []polytypes.Market{{
				ID:           "market-eth",
				Question:     "ETH up or down in 5m?",
				Active:       true,
				ClobTokenIDs: `["eth-up","eth-down"]`,
			}},
		},
		{
			ID:     "event-closed",
			Title:  "BTC 5m closed event",
			Active: true,
			Closed: true,
			Markets: []polytypes.Market{{
				ID:           "hidden",
				Question:     "BTC hidden 5m?",
				Active:       true,
				ClobTokenIDs: `["hidden"]`,
			}},
		},
	}}}
	fake := &fakeStreamer{onSubscribe: func(s *fakeStreamer) {
		s.handlers.OnBook(stream.BookMessage{EventType: "book", AssetID: "btc-up"})
	}}
	var emitted []interface{}

	err := New(searcher, func(config StreamConfig) Streamer {
		fake.config = config
		return fake
	}).Run(context.Background(), Request{
		Asset:          "BTC",
		Interval:       "5m",
		URL:            "wss://example.test/ws",
		MaxMessages:    1,
		CustomFeatures: true,
	}, func(v interface{}) { emitted = append(emitted, v) }, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if searcher.params == nil {
		t.Fatal("Search was not called")
	}
	if searcher.params.Q != "BTC 5m" {
		t.Fatalf("query=%q, want BTC 5m", searcher.params.Q)
	}
	if searcher.params.LimitPerType == nil || *searcher.params.LimitPerType != 50 {
		t.Fatalf("LimitPerType=%v, want 50", searcher.params.LimitPerType)
	}
	if fake.config.URL != "wss://example.test/ws" || !fake.config.CustomFeatures {
		t.Fatalf("unexpected stream config: %+v", fake.config)
	}
	if !fake.connected || !fake.closed {
		t.Fatalf("connected=%v closed=%v, want both true", fake.connected, fake.closed)
	}
	if want := []string{"btc-up", "btc-down"}; !reflect.DeepEqual(fake.subscribedTo, want) {
		t.Fatalf("subscribedTo=%v, want %v", fake.subscribedTo, want)
	}
	if len(emitted) != 2 {
		t.Fatalf("emitted=%d, want status plus one event: %+v", len(emitted), emitted)
	}
	status, ok := emitted[0].(Status)
	if !ok {
		t.Fatalf("first emission type %T, want Status", emitted[0])
	}
	if status.Status != "connecting" || status.Markets != 2 || status.Asset != "BTC" || status.Interval != "5m" || !reflect.DeepEqual(status.TokenIDs, []string{"btc-up", "btc-down"}) {
		t.Fatalf("unexpected status: %+v", status)
	}
	book, ok := emitted[1].(stream.BookMessage)
	if !ok || book.AssetID != "btc-up" {
		t.Fatalf("unexpected event: %#v", emitted[1])
	}
}

func TestRunnerEmitsStatsFromMarketStreamerWhenRequested(t *testing.T) {
	searcher := &fakeSearcher{resp: &polytypes.SearchResponse{Events: []polytypes.Event{{
		Title:  "BTC 5m event",
		Active: true,
		Markets: []polytypes.Market{{
			Question:     "BTC up or down in 5m?",
			Active:       true,
			ClobTokenIDs: `["btc-up"]`,
		}},
	}}}}
	fake := &fakeStreamer{onSubscribe: func(s *fakeStreamer) {
		s.handlers.OnBook(stream.BookMessage{EventType: "book", AssetID: "btc-up"})
	}}
	var emitted []interface{}

	err := New(searcher, func(config StreamConfig) Streamer { return fake }).Run(context.Background(), Request{MaxMessages: 1, Stats: true}, func(v interface{}) {
		emitted = append(emitted, v)
	}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(emitted) != 3 {
		t.Fatalf("emitted=%d, want status, event, stats: %+v", len(emitted), emitted)
	}
	if _, ok := emitted[2].(stream.StreamStatsSnapshot); !ok {
		t.Fatalf("third emission type %T, want stats", emitted[2])
	}
}

func TestRunnerDefaultsEmptyQueryToCrypto(t *testing.T) {
	searcher := &fakeSearcher{resp: &polytypes.SearchResponse{Events: []polytypes.Event{{
		Title:  "Any crypto",
		Active: true,
		Markets: []polytypes.Market{{
			Question:     "Any market",
			Active:       true,
			ClobTokenIDs: `token-1`,
		}},
	}}}}
	fake := &fakeStreamer{}

	err := New(searcher, func(config StreamConfig) Streamer { return fake }).Run(context.Background(), Request{}, func(v interface{}) {}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if searcher.params == nil || searcher.params.Q != "crypto" {
		t.Fatalf("query=%v, want crypto", searcher.params)
	}
}

func TestRunnerReturnsErrorWhenNoActiveCryptoTokensMatch(t *testing.T) {
	searcher := &fakeSearcher{resp: &polytypes.SearchResponse{Events: []polytypes.Event{{
		Title:  "ETH 5m event",
		Active: true,
		Markets: []polytypes.Market{{
			Question:     "ETH up or down in 5m?",
			Active:       true,
			ClobTokenIDs: `["eth-up"]`,
		}},
	}}}}
	var newStreamerCalled bool

	err := New(searcher, func(config StreamConfig) Streamer {
		newStreamerCalled = true
		return &fakeStreamer{}
	}).Run(context.Background(), Request{Asset: "BTC", Interval: "5m"}, func(v interface{}) {}, nil)
	if err == nil || err.Error() != "no active crypto markets found for asset=BTC interval=5m" {
		t.Fatalf("error=%v, want no active market error", err)
	}
	if newStreamerCalled {
		t.Fatal("streamer was created despite no matching tokens")
	}
}

func TestRunnerPropagatesSearchConnectSubscribeAndWaitErrors(t *testing.T) {
	searchErr := errors.New("gamma down")
	err := New(&fakeSearcher{err: searchErr}, func(config StreamConfig) Streamer { return &fakeStreamer{} }).Run(context.Background(), Request{}, func(v interface{}) {}, nil)
	if !errors.Is(err, searchErr) {
		t.Fatalf("search error=%v, want %v", err, searchErr)
	}

	baseSearcher := &fakeSearcher{resp: &polytypes.SearchResponse{Events: []polytypes.Event{{
		Title:  "BTC 5m event",
		Active: true,
		Markets: []polytypes.Market{{
			Question:     "BTC up or down in 5m?",
			Active:       true,
			ClobTokenIDs: `["btc-up"]`,
		}},
	}}}}
	for name, fake := range map[string]*fakeStreamer{
		"connect":   {connectErr: errors.New("connect failed")},
		"subscribe": {subscribeErr: errors.New("subscribe failed")},
		"wait":      {waitErr: errors.New("wait failed")},
	} {
		t.Run(name, func(t *testing.T) {
			err := New(baseSearcher, func(config StreamConfig) Streamer { return fake }).Run(context.Background(), Request{}, func(v interface{}) {}, nil)
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
	searcher := &fakeSearcher{resp: &polytypes.SearchResponse{Events: []polytypes.Event{{
		Title:  "BTC 5m event",
		Active: true,
		Markets: []polytypes.Market{{
			Question:     "BTC up or down in 5m?",
			Active:       true,
			ClobTokenIDs: `["btc-up"]`,
		}},
	}}}}
	streamErr := errors.New("read failed")
	fake := &fakeStreamer{onSubscribe: func(s *fakeStreamer) {
		s.handlers.OnError(streamErr)
	}}
	var reported []error

	err := New(searcher, func(config StreamConfig) Streamer { return fake }).Run(context.Background(), Request{MaxMessages: 0}, func(v interface{}) {}, func(err error) {
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
		s.handlers.OnBook(stream.BookMessage{EventType: "book", AssetID: "btc-up"})
	}}
	err = New(searcher, func(config StreamConfig) Streamer { return fake }).Run(context.Background(), Request{MaxMessages: 1}, func(v interface{}) {}, func(err error) {
		reported = append(reported, err)
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(reported) != 0 {
		t.Fatalf("reported=%v, want none for bounded stream", reported)
	}
}
