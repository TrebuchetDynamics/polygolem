// Package streamcrypto discovers crypto markets and streams their CLOB events without Cobra coupling.
package streamcrypto

import (
	"context"
	"fmt"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/cryptomarkets"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/streammarket"
)

const searchLimit = 50

// Searcher searches Gamma markets and events.
type Searcher interface {
	Search(ctx context.Context, params *polytypes.SearchParams) (*polytypes.SearchResponse, error)
}

// Request contains filters and streaming options for crypto streams.
type Request struct {
	Asset          string
	Interval       string
	URL            string
	MaxMessages    int
	CustomFeatures bool
	Stats          bool
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
	query, tokenIDs, err := r.discoverTokens(ctx, req)
	if err != nil {
		return err
	}
	_ = query
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
	return streammarket.New(r.newStreamer).Run(ctx, streammarket.Request{
		AssetIDs:       tokenIDs,
		URL:            req.URL,
		MaxMessages:    req.MaxMessages,
		CustomFeatures: req.CustomFeatures,
		Stats:          req.Stats,
	}, emit, reportError)
}

func (r *Runner) discoverTokens(ctx context.Context, req Request) (string, []string, error) {
	filter := cryptomarkets.Filter{Asset: req.Asset, Interval: req.Interval}
	query := cryptomarkets.Query(filter)
	limit := searchLimit
	resp, err := r.searcher.Search(ctx, &polytypes.SearchParams{
		Q:            query,
		LimitPerType: &limit,
	})
	if err != nil {
		return query, nil, err
	}

	var tokenIDs []string
	for _, candidate := range cryptomarkets.Select(resp, filter) {
		tokenIDs = append(tokenIDs, candidate.TokenIDs...)
	}
	return query, tokenIDs, nil
}

// NewInternalStreamer creates a Streamer backed by the internal WebSocket client.
func NewInternalStreamer(config StreamConfig) Streamer {
	return streammarket.NewInternalStreamer(config)
}
