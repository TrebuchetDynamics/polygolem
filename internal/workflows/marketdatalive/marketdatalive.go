// Package marketdatalive streams enriched market-data snapshots without Cobra coupling.
package marketdatalive

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/marketdata"
	"github.com/TrebuchetDynamics/polygolem/pkg/stream"
)

// Request contains asset IDs and streaming options for live market-data snapshots.
type Request struct {
	AssetIDs       []string
	AssetIDsRaw    string
	URL            string
	MaxMessages    int
	CustomFeatures bool
	Level          int
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
	OnError          func(error)
}

// Streamer is the streaming seam used by Runner.
type Streamer interface {
	SetHandlers(Handlers)
	Connect(ctx context.Context) error
	SubscribeAssets(ctx context.Context, assetIDs []string) error
	Wait(ctx context.Context, done <-chan struct{}) error
	Close()
}

// StreamFactory creates a stream adapter for one run.
type StreamFactory func(StreamConfig) Streamer

// EmitFunc receives enriched market-data snapshots.
type EmitFunc func(interface{})

// ErrorFunc receives asynchronous stream errors for unbounded streams.
type ErrorFunc func(error)

// Runner owns live market-data stream orchestration.
type Runner struct {
	newStreamer StreamFactory
}

// New creates a live market-data runner.
func New(newStreamer StreamFactory) *Runner {
	if newStreamer == nil {
		newStreamer = NewSDKStreamer
	}
	return &Runner{newStreamer: newStreamer}
}

// Run connects, subscribes to asset IDs, tracks raw stream events, and emits snapshots.
func (r *Runner) Run(ctx context.Context, req Request, emit EmitFunc, reportError ErrorFunc) error {
	assetIDs := req.assetIDs()
	if len(assetIDs) == 0 {
		return fmt.Errorf("--asset-ids required")
	}
	if emit == nil {
		emit = func(interface{}) {}
	}

	tracker := marketdata.NewTracker()
	done := make(chan struct{})
	var closeDone sync.Once
	count := 0
	emitSnapshot := func(snapshot marketdata.Snapshot) {
		if req.MaxMessages > 0 && count >= req.MaxMessages {
			return
		}
		count++
		emit(snapshot)
		if req.MaxMessages > 0 && count >= req.MaxMessages {
			closeDone.Do(func() { close(done) })
		}
	}

	streamer := r.newStreamer(StreamConfig{URL: req.URL, CustomFeatures: req.CustomFeatures, Level: req.Level})
	streamer.SetHandlers(Handlers{
		OnBook: func(msg stream.BookMessage) {
			emitSnapshot(tracker.ApplyBook(msg))
		},
		OnPriceChange: func(msg stream.PriceChangeMessage) {
			for _, snapshot := range tracker.ApplyPriceChange(msg) {
				emitSnapshot(snapshot)
			}
		},
		OnLastTrade: func(msg stream.LastTradeMessage) {
			emitSnapshot(tracker.ApplyLastTrade(msg))
		},
		OnBestBidAsk: func(msg stream.BestBidAskMessage) {
			emitSnapshot(tracker.ApplyBestBidAsk(msg))
		},
		OnTickSizeChange: func(msg stream.TickSizeChangeMessage) {
			emitSnapshot(tracker.ApplyTickSizeChange(msg))
		},
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
	return streamer.Wait(ctx, done)
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

type sdkStreamer struct {
	client *stream.MarketClient
}

// NewSDKStreamer creates a Streamer backed by the public stream SDK.
func NewSDKStreamer(config StreamConfig) Streamer {
	cfg := stream.DefaultConfig(config.URL)
	cfg.PingInterval = 10 * time.Second
	cfg.CustomFeatureEnabled = config.CustomFeatures
	cfg.Level = config.Level
	return &sdkStreamer{client: stream.NewMarketClient(cfg)}
}

func (s *sdkStreamer) SetHandlers(handlers Handlers) {
	s.client.OnBook = handlers.OnBook
	s.client.OnPriceChange = handlers.OnPriceChange
	s.client.OnLastTrade = handlers.OnLastTrade
	s.client.OnTickSizeChange = handlers.OnTickSizeChange
	s.client.OnBestBidAsk = handlers.OnBestBidAsk
	s.client.OnError = handlers.OnError
}

func (s *sdkStreamer) Connect(ctx context.Context) error {
	return s.client.Connect(ctx)
}

func (s *sdkStreamer) SubscribeAssets(ctx context.Context, assetIDs []string) error {
	return s.client.SubscribeAssets(ctx, assetIDs)
}

func (s *sdkStreamer) Wait(ctx context.Context, done <-chan struct{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (s *sdkStreamer) Close() {
	s.client.Close()
}
