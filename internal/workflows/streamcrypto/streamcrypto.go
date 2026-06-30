// Package streamcrypto discovers crypto markets and streams their CLOB events without Cobra coupling.
package streamcrypto

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/stream"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/cryptomarkets"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/streammarket"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
)

const searchLimit = 50

// Searcher searches Gamma markets and events.
type Searcher = cryptomarkets.Searcher

// EventFetcher fetches a Gamma event by deterministic slug.
type EventFetcher interface {
	EventBySlug(ctx context.Context, slug string) (*polytypes.Event, error)
}

// Request contains filters and streaming options for crypto streams.
type Request struct {
	Asset          string
	Interval       string
	URL            string
	MaxMessages    int
	CustomFeatures bool
	Stats          bool
	// RefreshInterval overrides boundary refresh timing for tests. Zero uses the
	// next crypto interval boundary; refreshes only run for unbounded streams.
	RefreshInterval time.Duration
}

// Status is emitted before connecting to the market stream.
type Status struct {
	Status   string   `json:"status"`
	Markets  int      `json:"markets"`
	Asset    string   `json:"asset"`
	Interval string   `json:"interval"`
	TokenIDs []string `json:"token_ids"`
}

// StreamConfig contains the stream options controlled by this workflow.
type StreamConfig = streammarket.StreamConfig

// Handlers contains event callbacks used by a Streamer adapter.
type Handlers = streammarket.Handlers

// Streamer is the streaming seam used by Runner.
type Streamer = streammarket.Streamer

// StreamFactory creates a stream adapter for one run.
type StreamFactory = streammarket.StreamFactory

// EmitFunc receives status and stream events.
type EmitFunc = streammarket.EmitFunc

// ErrorFunc receives asynchronous stream errors for unbounded streams.
type ErrorFunc = streammarket.ErrorFunc

// Runner owns crypto stream discovery and streaming orchestration.
type Runner struct {
	searcher    Searcher
	newStreamer StreamFactory
	Now         func() time.Time
}

// New creates a crypto stream runner.
func New(searcher Searcher, newStreamer StreamFactory) *Runner {
	if newStreamer == nil {
		newStreamer = NewInternalStreamer
	}
	return &Runner{searcher: searcher, newStreamer: newStreamer}
}

// Run discovers active crypto token IDs, emits a connection status, then streams events.
func (r *Runner) Run(ctx context.Context, req Request, emit EmitFunc, reportError ErrorFunc) error {
	if emit == nil {
		emit = func(interface{}) {}
	}
	tokenIDs, err := r.tokenIDs(ctx, req)
	if err != nil {
		return err
	}
	if len(tokenIDs) == 0 {
		return fmt.Errorf("no active crypto markets found for asset=%s interval=%s", req.Asset, req.Interval)
	}

	emit(Status{
		Status:   "connecting",
		Markets:  len(tokenIDs),
		Asset:    req.Asset,
		Interval: req.Interval,
		TokenIDs: tokenIDs,
	})

	done := make(chan struct{})
	var closeDone sync.Once
	var mu sync.Mutex
	count := 0
	emitEvent := func(v interface{}) {
		mu.Lock()
		defer mu.Unlock()
		if req.MaxMessages > 0 && count >= req.MaxMessages {
			return
		}
		count++
		emit(v)
		if req.MaxMessages > 0 && count >= req.MaxMessages {
			closeDone.Do(func() { close(done) })
		}
	}

	streamer := r.newStreamer(StreamConfig{URL: req.URL, CustomFeatures: req.CustomFeatures})
	streamer.SetHandlers(Handlers{
		OnBook:           func(msg stream.BookMessage) { emitEvent(msg) },
		OnPriceChange:    func(msg stream.PriceChangeMessage) { emitEvent(msg) },
		OnLastTrade:      func(msg stream.LastTradeMessage) { emitEvent(msg) },
		OnTickSizeChange: func(msg stream.TickSizeChangeMessage) { emitEvent(msg) },
		OnBestBidAsk:     func(msg stream.BestBidAskMessage) { emitEvent(msg) },
		OnNewMarket:      func(msg stream.NewMarketMessage) { emitEvent(msg) },
		OnMarketResolved: func(msg stream.MarketResolvedMessage) { emitEvent(msg) },
		OnError: func(err error) {
			if req.MaxMessages == 0 && reportError != nil {
				reportError(err)
			}
		},
	})
	if err := streamer.Connect(ctx); err != nil {
		return err
	}
	defer streamer.Close()
	if err := streamer.SubscribeAssets(ctx, tokenIDs); err != nil {
		return err
	}
	if req.MaxMessages == 0 {
		go r.refreshLoop(ctx, req, streamer, reportError)
	}
	err = streamer.Wait(ctx, done)
	if req.Stats {
		emit(streamer.Stats())
	}
	return err
}

func (r *Runner) refreshLoop(ctx context.Context, req Request, streamer Streamer, reportError ErrorFunc) {
	if _, ok := r.searcher.(EventFetcher); !ok || strings.TrimSpace(req.Asset) == "" || strings.TrimSpace(req.Interval) == "" {
		return
	}
	for {
		delay := req.RefreshInterval
		if delay <= 0 {
			step := windowStep(strings.TrimSpace(req.Interval))
			if step == 0 {
				return
			}
			delay = nextBoundaryDelay(r.now(), step)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		ids, err := r.tokenIDs(ctx, req)
		if err != nil {
			if reportError != nil {
				reportError(err)
			}
			continue
		}
		if len(ids) == 0 {
			continue
		}
		if err := streamer.SubscribeAssets(ctx, ids); err != nil && reportError != nil {
			reportError(err)
		}
	}
}

func (r *Runner) tokenIDs(ctx context.Context, req Request) ([]string, error) {
	if ids := r.lookaheadTokenIDs(ctx, req); len(ids) > 0 {
		return ids, nil
	}
	_, candidates, err := cryptomarkets.Discover(ctx, r.searcher, cryptomarkets.Filter{Asset: req.Asset, Interval: req.Interval}, searchLimit)
	if err != nil {
		return nil, err
	}
	return cryptomarkets.TokenIDs(candidates), nil
}

func (r *Runner) lookaheadTokenIDs(ctx context.Context, req Request) []string {
	asset := strings.TrimSpace(req.Asset)
	interval := strings.TrimSpace(req.Interval)
	fetcher, ok := r.searcher.(EventFetcher)
	if !ok || asset == "" || interval == "" {
		return nil
	}
	start, err := windowStartAt(interval, r.now())
	if err != nil {
		return nil
	}
	var ids []string
	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		slug := marketresolver.CryptoWindowSlug(asset, interval, start.Add(time.Duration(i)*windowStep(interval)))
		if slug == "" {
			continue
		}
		event, err := fetcher.EventBySlug(ctx, slug)
		if err != nil || event == nil {
			continue
		}
		for _, market := range event.Markets {
			if !market.Active || market.Closed {
				continue
			}
			for _, id := range cryptomarkets.ParseTokenIDs(market.ClobTokenIDs) {
				if id != "" && !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
		}
	}
	return ids
}

func (r *Runner) now() time.Time {
	if r.Now == nil {
		return time.Now().UTC()
	}
	return r.Now()
}

func windowStartAt(interval string, now time.Time) (time.Time, error) {
	step := windowStep(interval)
	if step == 0 {
		return time.Time{}, fmt.Errorf("unsupported interval: %s (use 5m, 15m, 1h, 4h)", interval)
	}
	unix := now.UTC().Unix()
	seconds := int64(step / time.Second)
	return time.Unix(unix-(unix%seconds), 0).UTC(), nil
}

func nextBoundaryDelay(now time.Time, step time.Duration) time.Duration {
	start := now.UTC().Truncate(step)
	next := start.Add(step)
	delay := next.Sub(now.UTC())
	if delay <= 0 {
		return step
	}
	return delay
}

func windowStep(interval string) time.Duration {
	switch interval {
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "1h":
		return time.Hour
	case "4h":
		return 4 * time.Hour
	}
	return 0
}

// NewInternalStreamer creates a Streamer backed by the internal WebSocket client.
func NewInternalStreamer(config StreamConfig) Streamer {
	return streammarket.NewInternalStreamer(config)
}
