// Package streammarket streams public CLOB market events without Cobra coupling.
package streammarket

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/stream"
)

// Request contains asset IDs and streaming options for public market streams.
type Request struct {
	AssetIDs       []string
	AssetIDsRaw    string
	URL            string
	MaxMessages    int
	CustomFeatures bool
	Level          int
	Stats          bool
}

// StreamConfig contains the stream options controlled by this workflow.
type StreamConfig struct {
	URL            string
	CustomFeatures bool
	Level          int
}

// Handlers contains event callbacks used by a Streamer adapter.
type Handlers struct {
	OnBook           func(stream.BookMessage)
	OnPriceChange    func(stream.PriceChangeMessage)
	OnLastTrade      func(stream.LastTradeMessage)
	OnTickSizeChange func(stream.TickSizeChangeMessage)
	OnBestBidAsk     func(stream.BestBidAskMessage)
	OnNewMarket      func(stream.NewMarketMessage)
	OnMarketResolved func(stream.MarketResolvedMessage)
	OnError          func(error)
}

// Streamer is the streaming seam used by Runner.
type Streamer interface {
	SetHandlers(Handlers)
	Connect(ctx context.Context) error
	SubscribeAssets(ctx context.Context, assetIDs []string) error
	Wait(ctx context.Context, done <-chan struct{}) error
	Close()
	Stats() stream.StreamStatsSnapshot
}

// StreamFactory creates a stream adapter for one run.
type StreamFactory func(StreamConfig) Streamer

// EmitFunc receives stream events.
type EmitFunc func(interface{})

// ErrorFunc receives asynchronous stream errors for unbounded streams.
type ErrorFunc func(error)

// Runner owns public market stream orchestration.
type Runner struct {
	newStreamer StreamFactory
}

// New creates a public market stream runner.
func New(newStreamer StreamFactory) *Runner {
	if newStreamer == nil {
		newStreamer = NewInternalStreamer
	}
	return &Runner{newStreamer: newStreamer}
}

// Run connects, subscribes to asset IDs, and emits stream events until stopped.
func (r *Runner) Run(ctx context.Context, req Request, emit EmitFunc, reportError ErrorFunc) error {
	assetIDs := req.assetIDs()
	if len(assetIDs) == 0 {
		return fmt.Errorf("--asset-ids required")
	}
	if emit == nil {
		emit = func(interface{}) {}
	}

	done := make(chan struct{})
	var closeDone sync.Once
	count := 0
	emitEvent := func(v interface{}) {
		if req.MaxMessages > 0 && count >= req.MaxMessages {
			return
		}
		count++
		emit(v)
		if req.MaxMessages > 0 && count >= req.MaxMessages {
			closeDone.Do(func() { close(done) })
		}
	}

	streamer := r.newStreamer(StreamConfig{URL: req.URL, CustomFeatures: req.CustomFeatures, Level: req.Level})
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
	if err := streamer.SubscribeAssets(ctx, assetIDs); err != nil {
		return err
	}
	err := streamer.Wait(ctx, done)
	if req.Stats {
		emit(streamer.Stats())
	}
	return err
}

func (r Request) assetIDs() []string {
	if len(r.AssetIDs) > 0 {
		return append([]string(nil), r.AssetIDs...)
	}
	return splitCSV(r.AssetIDsRaw)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

type internalStreamer struct {
	client *stream.MarketClient
}

// NewInternalStreamer creates a Streamer backed by the internal WebSocket client.
func NewInternalStreamer(config StreamConfig) Streamer {
	cfg := stream.DefaultConfig(config.URL)
	cfg.PingInterval = 10 * time.Second
	cfg.CustomFeatureEnabled = config.CustomFeatures
	cfg.Level = config.Level
	return &internalStreamer{client: stream.NewMarketClient(cfg)}
}

func (s *internalStreamer) SetHandlers(handlers Handlers) {
	s.client.OnBook = handlers.OnBook
	s.client.OnPriceChange = handlers.OnPriceChange
	s.client.OnLastTrade = handlers.OnLastTrade
	s.client.OnTickSizeChange = handlers.OnTickSizeChange
	s.client.OnBestBidAsk = handlers.OnBestBidAsk
	s.client.OnNewMarket = handlers.OnNewMarket
	s.client.OnMarketResolved = handlers.OnMarketResolved
	s.client.OnError = handlers.OnError
}

func (s *internalStreamer) Connect(ctx context.Context) error {
	return s.client.Connect(ctx)
}

func (s *internalStreamer) SubscribeAssets(ctx context.Context, assetIDs []string) error {
	return s.client.SubscribeAssets(ctx, assetIDs)
}

func (s *internalStreamer) Wait(ctx context.Context, done <-chan struct{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (s *internalStreamer) Close() {
	s.client.Close()
}

func (s *internalStreamer) Stats() stream.StreamStatsSnapshot {
	return s.client.Stats()
}
