// Package discoverreads owns generic read-only market discovery behavior without Cobra coupling.
package discoverreads

import (
	"context"
	"fmt"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	sdkgamma "github.com/TrebuchetDynamics/polygolem/pkg/gamma"
)

// Operation selects one generic read-only discovery request.
type Operation string

const (
	Markets        Operation = "markets"
	Search         Operation = "search"
	Market         Operation = "market"
	Enrich         Operation = "enrich"
	Tags           Operation = "tags"
	Series         Operation = "series"
	Comments       Operation = "comments"
	Categories     Operation = "categories"
	CategoryEvents Operation = "category-events"
)

// Config wires discovery read adapters.
type Config struct {
	Gamma    GammaReader
	Enricher Enricher
}

// Request contains one generic read-only discovery request.
type Request struct {
	Operation Operation

	Query  string
	Limit  int
	Offset int
	Order  string

	ID   string
	Slug string

	Active    bool
	Closed    bool
	Ascending bool
	TagID     int
	Cursor    string

	EntityID   int
	EntityType string
	User       string
}

// GammaReader is the Gamma read adapter used by this workflow.
type GammaReader interface {
	Markets(context.Context, *polytypes.GetMarketsParams) ([]polytypes.Market, error)
	MarketByID(context.Context, string) (*polytypes.Market, error)
	MarketBySlug(context.Context, string) (*polytypes.Market, error)
	Search(context.Context, *polytypes.SearchParams) (*polytypes.SearchResponse, error)
	Tags(context.Context, *polytypes.GetTagsParams) ([]polytypes.Tag, error)
	TagByID(context.Context, string) (*polytypes.Tag, error)
	TagBySlug(context.Context, string) (*polytypes.Tag, error)
	Series(context.Context, *polytypes.GetSeriesParams) ([]polytypes.Series, error)
	SeriesByID(context.Context, string) (*polytypes.Series, error)
	Comments(context.Context, *polytypes.CommentQuery) ([]polytypes.Comment, error)
	CommentByID(context.Context, string) (*polytypes.Comment, error)
	CommentsByUser(context.Context, string, int) ([]polytypes.Comment, error)
	CategoryEvents(context.Context, polytypes.PolymarketCategory, *polytypes.CategoryEventsParams) ([]polytypes.Event, string, error)
}

// Enricher is the CLOB-backed market enrichment adapter used by this workflow.
type Enricher interface {
	EnrichMarket(context.Context, polytypes.Market) (*polytypes.EnrichedMarket, error)
}

// Runner executes generic read-only discovery requests.
type Runner struct {
	gamma    GammaReader
	enricher Enricher
}

// New creates a discovery reads workflow runner.
func New(cfg Config) *Runner {
	return &Runner{gamma: cfg.Gamma, enricher: cfg.Enricher}
}

// Run executes req and returns the command payload shape.
func (r *Runner) Run(ctx context.Context, req Request) (any, error) {
	switch req.Operation {
	case Markets:
		return r.runMarkets(ctx, req)
	case Search:
		return r.runSearch(ctx, req)
	case Market:
		return r.runMarket(ctx, req)
	case Enrich:
		return r.runEnrich(ctx, req)
	case Tags:
		return r.runTags(ctx, req)
	case Series:
		return r.runSeries(ctx, req)
	case Comments:
		return r.runComments(ctx, req)
	case Categories:
		return r.runCategories(req)
	case CategoryEvents:
		return r.runCategoryEvents(ctx, req)
	default:
		return nil, fmt.Errorf("unknown discover operation %q", req.Operation)
	}
}

func (r *Runner) runMarkets(ctx context.Context, req Request) ([]polytypes.Market, error) {
	params := &polytypes.GetMarketsParams{
		Limit:     req.Limit,
		Offset:    req.Offset,
		Order:     req.Order,
		Active:    &req.Active,
		Closed:    &req.Closed,
		Ascending: &req.Ascending,
	}
	if req.TagID > 0 {
		params.TagID = &req.TagID
	}
	return r.gamma.Markets(ctx, params)
}

func (r *Runner) runSearch(ctx context.Context, req Request) (any, error) {
	if req.Query != "" {
		return r.gamma.Search(ctx, &polytypes.SearchParams{Q: req.Query, LimitPerType: &req.Limit})
	}
	active, closed := true, false
	return r.gamma.Markets(ctx, &polytypes.GetMarketsParams{Active: &active, Closed: &closed, Limit: req.Limit})
}

func (r *Runner) runMarket(ctx context.Context, req Request) (*polytypes.Market, error) {
	if req.ID != "" {
		return r.gamma.MarketByID(ctx, req.ID)
	}
	if req.Slug != "" {
		return r.gamma.MarketBySlug(ctx, req.Slug)
	}
	return nil, fmt.Errorf("--id or --slug required")
}

func (r *Runner) runEnrich(ctx context.Context, req Request) (*polytypes.EnrichedMarket, error) {
	if req.ID == "" {
		return nil, fmt.Errorf("--id required")
	}
	m, err := r.gamma.MarketByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, fmt.Errorf("market %q not found", req.ID)
	}
	return r.enricher.EnrichMarket(ctx, *m)
}

func (r *Runner) runTags(ctx context.Context, req Request) (any, error) {
	if req.ID != "" {
		return r.gamma.TagByID(ctx, req.ID)
	}
	if req.Slug != "" {
		return r.gamma.TagBySlug(ctx, req.Slug)
	}
	return r.gamma.Tags(ctx, &polytypes.GetTagsParams{Limit: req.Limit, Offset: req.Offset})
}

func (r *Runner) runSeries(ctx context.Context, req Request) (any, error) {
	if req.ID != "" {
		return r.gamma.SeriesByID(ctx, req.ID)
	}
	return r.gamma.Series(ctx, &polytypes.GetSeriesParams{Limit: req.Limit, Offset: req.Offset, Closed: &req.Closed})
}

func (r *Runner) runCategories(req Request) (any, error) {
	if req.Slug != "" {
		category, ok := sdkgamma.PolymarketCategoryBySlug(req.Slug)
		if !ok {
			return nil, fmt.Errorf("unknown polymarket category %q", req.Slug)
		}
		return category, nil
	}
	return sdkgamma.PolymarketCategories(), nil
}

func (r *Runner) runCategoryEvents(ctx context.Context, req Request) (*polytypes.CategoryEventsResponse, error) {
	if req.Slug == "" {
		return nil, fmt.Errorf("--slug required")
	}
	category, ok := sdkgamma.PolymarketCategoryBySlug(req.Slug)
	if !ok {
		return nil, fmt.Errorf("unknown polymarket category %q", req.Slug)
	}
	if category.FeedMode == sdkgamma.CategoryFeedRouteOnly {
		return nil, fmt.Errorf("category %q is route-only and has no Gamma events/keyset feed", category.Slug)
	}
	closed := req.Closed
	params := &polytypes.CategoryEventsParams{Limit: req.Limit, Cursor: req.Cursor, Order: req.Order, Closed: &closed}
	if req.Ascending {
		params.Ascending = &req.Ascending
	}
	events, cursor, err := r.gamma.CategoryEvents(ctx, category, params)
	if err != nil {
		return nil, err
	}
	return &polytypes.CategoryEventsResponse{Category: category, Events: events, NextCursor: cursor, HasMore: cursor != ""}, nil
}

func (r *Runner) runComments(ctx context.Context, req Request) (any, error) {
	if req.ID != "" {
		return r.gamma.CommentByID(ctx, req.ID)
	}
	if req.User != "" {
		return r.gamma.CommentsByUser(ctx, req.User, req.Limit)
	}
	params := &polytypes.CommentQuery{Limit: req.Limit, Offset: req.Offset}
	if req.EntityID > 0 {
		params.EntityID = &req.EntityID
	}
	if req.EntityType != "" {
		params.EntityType = &req.EntityType
	}
	return r.gamma.Comments(ctx, params)
}
