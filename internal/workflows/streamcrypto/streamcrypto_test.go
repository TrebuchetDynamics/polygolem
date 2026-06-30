package streamcrypto

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/stream"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
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

type fakeLookaheadSearcher struct {
	events map[string]*polytypes.Event
	slugs  []string
}

func (f *fakeLookaheadSearcher) Search(ctx context.Context, params *polytypes.SearchParams) (*polytypes.SearchResponse, error) {
	return nil, errors.New("unexpected search fallback")
}

func (f *fakeLookaheadSearcher) EventBySlug(ctx context.Context, slug string) (*polytypes.Event, error) {
	f.slugs = append(f.slugs, slug)
	return f.events[slug], nil
}

type fakeStreamer struct {
	config        StreamConfig
	handlers      Handlers
	connected     bool
	closed        bool
	subscribedTo  []string
	subscriptions [][]string
	connectErr    error
	subscribeErr  error
	waitErr       error
	waitUntil     <-chan struct{}
	onSubscribe   func(*fakeStreamer)
}

func (f *fakeStreamer) SetHandlers(handlers Handlers) { f.handlers = handlers }
func (f *fakeStreamer) Connect(ctx context.Context) error {
	f.connected = true
	return f.connectErr
}
func (f *fakeStreamer) SubscribeAssets(ctx context.Context, assetIDs []string) error {
	f.subscribedTo = append([]string(nil), assetIDs...)
	f.subscriptions = append(f.subscriptions, append([]string(nil), assetIDs...))
	if f.onSubscribe != nil {
		f.onSubscribe(f)
	}
	return f.subscribeErr
}
func (f *fakeStreamer) Wait(ctx context.Context, done <-chan struct{}) error {
	if f.waitUntil != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			return nil
		case <-f.waitUntil:
			return f.waitErr
		}
	}
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
	if searcher.params.Q != "bitcoin 5m updown" {
		t.Fatalf("query=%q, want bitcoin 5m updown", searcher.params.Q)
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

func TestRunnerRefreshesLookaheadTokensAtBoundary(t *testing.T) {
	initialNow := time.Date(2026, 5, 23, 12, 34, 56, 0, time.UTC)
	refreshNow := time.Date(2026, 5, 23, 12, 35, 1, 0, time.UTC)
	initialStart := time.Date(2026, 5, 23, 12, 30, 0, 0, time.UTC)
	refreshStart := time.Date(2026, 5, 23, 12, 35, 0, 0, time.UTC)
	searcher := &fakeLookaheadSearcher{events: map[string]*polytypes.Event{}}
	for i, pair := range [][]string{{"cur-up", "cur-down"}, {"next-up", "next-down"}, {"later-up", "later-down"}, {"new-up", "new-down"}} {
		slug := marketresolver.CryptoWindowSlug("BTC", "5m", initialStart.Add(time.Duration(i)*5*time.Minute))
		searcher.events[slug] = &polytypes.Event{Active: true, Markets: []polytypes.Market{{Active: true, ClobTokenIDs: `["` + pair[0] + `","` + pair[1] + `"]`}}}
	}
	refreshed := make(chan struct{})
	fake := &fakeStreamer{waitUntil: refreshed}
	fake.onSubscribe = func(s *fakeStreamer) {
		if len(s.subscriptions) == 2 {
			close(refreshed)
		}
	}
	runner := New(searcher, func(config StreamConfig) Streamer { return fake })
	calls := 0
	runner.Now = func() time.Time {
		calls++
		if calls <= 1 {
			return initialNow
		}
		return refreshNow
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := runner.Run(ctx, Request{Asset: "BTC", Interval: "5m", RefreshInterval: time.Millisecond}, func(v interface{}) {}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(fake.subscriptions) != 2 {
		t.Fatalf("subscriptions=%v, want initial plus refresh", fake.subscriptions)
	}
	wantRefresh := []string{"next-up", "next-down", "later-up", "later-down", "new-up", "new-down"}
	if !reflect.DeepEqual(fake.subscriptions[1], wantRefresh) {
		t.Fatalf("refresh subscription=%v, want %v", fake.subscriptions[1], wantRefresh)
	}
	if wantFirstRefreshSlug := marketresolver.CryptoWindowSlug("BTC", "5m", refreshStart); searcher.slugs[3] != wantFirstRefreshSlug {
		t.Fatalf("first refresh slug=%q, want %q", searcher.slugs[3], wantFirstRefreshSlug)
	}
}

func TestRunnerSubscribesCurrentAndLookaheadWindowTokens(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 34, 56, 0, time.UTC)
	windowStart := time.Date(2026, 5, 23, 12, 30, 0, 0, time.UTC)
	searcher := &fakeLookaheadSearcher{events: map[string]*polytypes.Event{}}
	for i, pair := range [][]string{{"cur-up", "cur-down"}, {"next-up", "next-down"}, {"later-up", "later-down"}} {
		slug := marketresolver.CryptoWindowSlug("BTC", "5m", windowStart.Add(time.Duration(i)*5*time.Minute))
		searcher.events[slug] = &polytypes.Event{Active: true, Markets: []polytypes.Market{{
			Active:       true,
			ClobTokenIDs: `["` + pair[0] + `","` + pair[1] + `"]`,
		}}}
	}
	fake := &fakeStreamer{onSubscribe: func(s *fakeStreamer) {
		s.handlers.OnBook(stream.BookMessage{EventType: "book", AssetID: "cur-up"})
	}}
	var emitted []interface{}
	runner := New(searcher, func(config StreamConfig) Streamer { return fake })
	runner.Now = func() time.Time { return now }

	err := runner.Run(context.Background(), Request{Asset: "BTC", Interval: "5m", MaxMessages: 1}, func(v interface{}) {
		emitted = append(emitted, v)
	}, nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	want := []string{"cur-up", "cur-down", "next-up", "next-down", "later-up", "later-down"}
	if !reflect.DeepEqual(fake.subscribedTo, want) {
		t.Fatalf("subscribedTo=%v, want %v", fake.subscribedTo, want)
	}
	if len(searcher.slugs) != 3 {
		t.Fatalf("slugs=%v, want 3 window lookups", searcher.slugs)
	}
	status, ok := emitted[0].(Status)
	if !ok || status.Markets != len(want) || !reflect.DeepEqual(status.TokenIDs, want) {
		t.Fatalf("unexpected status: %+v", emitted[0])
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
