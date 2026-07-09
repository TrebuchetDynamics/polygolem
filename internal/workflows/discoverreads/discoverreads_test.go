package discoverreads

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

type fakeGamma struct {
	marketsCalled       bool
	searchParams        *polytypes.SearchParams
	marketByID          string
	marketByIDNil       bool
	categoryEventsCat   polytypes.PolymarketCategory
	categoryEventsParam *polytypes.CategoryEventsParams
}

func (f *fakeGamma) Markets(context.Context, *polytypes.GetMarketsParams) ([]polytypes.Market, error) {
	f.marketsCalled = true
	return []polytypes.Market{{ID: "market-1", Slug: "market-one"}}, nil
}

func (f *fakeGamma) MarketByID(_ context.Context, id string) (*polytypes.Market, error) {
	f.marketByID = id
	if f.marketByIDNil {
		return nil, nil // e.g. Gamma returned a JSON null body
	}
	return &polytypes.Market{ID: id, Slug: "market-one"}, nil
}

func (f *fakeGamma) MarketBySlug(context.Context, string) (*polytypes.Market, error) {
	return &polytypes.Market{ID: "market-1", Slug: "market-one"}, nil
}

func (f *fakeGamma) Search(_ context.Context, params *polytypes.SearchParams) (*polytypes.SearchResponse, error) {
	f.searchParams = params
	return &polytypes.SearchResponse{Events: []polytypes.Event{{ID: "event-1", Slug: "event-one"}}}, nil
}

func (f *fakeGamma) Tags(context.Context, *polytypes.GetTagsParams) ([]polytypes.Tag, error) {
	return []polytypes.Tag{{ID: "1", Label: "Politics"}}, nil
}

func (f *fakeGamma) TagByID(context.Context, string) (*polytypes.Tag, error) {
	return &polytypes.Tag{ID: "1", Label: "Politics"}, nil
}

func (f *fakeGamma) TagBySlug(context.Context, string) (*polytypes.Tag, error) {
	return &polytypes.Tag{ID: "1", Slug: "politics"}, nil
}

func (f *fakeGamma) Series(context.Context, *polytypes.GetSeriesParams) ([]polytypes.Series, error) {
	return []polytypes.Series{{ID: "series-1", Slug: "series-one"}}, nil
}

func (f *fakeGamma) SeriesByID(context.Context, string) (*polytypes.Series, error) {
	return &polytypes.Series{ID: "series-1", Slug: "series-one"}, nil
}

func (f *fakeGamma) Comments(context.Context, *polytypes.CommentQuery) ([]polytypes.Comment, error) {
	return []polytypes.Comment{{ID: "comment-1", Body: "hello"}}, nil
}

func (f *fakeGamma) CommentByID(context.Context, string) (*polytypes.Comment, error) {
	return &polytypes.Comment{ID: "comment-1", Body: "hello"}, nil
}

func (f *fakeGamma) CommentsByUser(context.Context, string, int) ([]polytypes.Comment, error) {
	return []polytypes.Comment{{ID: "comment-1", Body: "hello"}}, nil
}

func (f *fakeGamma) CategoryEvents(_ context.Context, category polytypes.PolymarketCategory, params *polytypes.CategoryEventsParams) ([]polytypes.Event, string, error) {
	f.categoryEventsCat = category
	f.categoryEventsParam = params
	return []polytypes.Event{{ID: "event-1", Slug: "event-one"}}, "cursor-2", nil
}

type fakeEnricher struct {
	marketID string
}

func (f *fakeEnricher) EnrichMarket(_ context.Context, market polytypes.Market) (*polytypes.EnrichedMarket, error) {
	f.marketID = market.ID
	return &polytypes.EnrichedMarket{Market: market}, nil
}

func TestRunnerSearchWithQueryUsesGammaSearch(t *testing.T) {
	gamma := &fakeGamma{}
	runner := New(Config{Gamma: gamma, Enricher: &fakeEnricher{}})

	got, err := runner.Run(context.Background(), Request{Operation: Search, Query: "btc", Limit: 5})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	resp, ok := got.(*polytypes.SearchResponse)
	if !ok || len(resp.Events) != 1 || resp.Events[0].Slug != "event-one" {
		t.Fatalf("result=%#v", got)
	}
	if gamma.searchParams == nil || gamma.searchParams.Q != "btc" || gamma.searchParams.LimitPerType == nil || *gamma.searchParams.LimitPerType != 5 {
		t.Fatalf("search params=%+v", gamma.searchParams)
	}
	if gamma.marketsCalled {
		t.Fatal("markets fallback should not run when query is set")
	}
}

func TestRunnerMarketRequiresIDOrSlugBeforeRead(t *testing.T) {
	gamma := &fakeGamma{}
	runner := New(Config{Gamma: gamma, Enricher: &fakeEnricher{}})

	_, err := runner.Run(context.Background(), Request{Operation: Market})
	if err == nil || !strings.Contains(err.Error(), "--id or --slug required") {
		t.Fatalf("error=%v, want --id or --slug required", err)
	}
	if gamma.marketByID != "" || gamma.marketsCalled || gamma.searchParams != nil {
		t.Fatalf("reader should not be called before market validation: %+v", gamma)
	}
}

func TestRunnerEnrichFetchesMarketThenEnriches(t *testing.T) {
	gamma := &fakeGamma{}
	enricher := &fakeEnricher{}
	runner := New(Config{Gamma: gamma, Enricher: enricher})

	got, err := runner.Run(context.Background(), Request{Operation: Enrich, ID: "market-1"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	enriched, ok := got.(*polytypes.EnrichedMarket)
	if !ok || enriched.Market.ID != "market-1" {
		t.Fatalf("result=%#v", got)
	}
	if gamma.marketByID != "market-1" || enricher.marketID != "market-1" {
		t.Fatalf("marketByID=%q enriched=%q", gamma.marketByID, enricher.marketID)
	}
}

// TestRunnerEnrichErrorsWhenMarketNotFound guards the regression where a nil
// market (Gamma returning a JSON null body) was dereferenced, panicking.
func TestRunnerCategoriesReturnsCuratedMapping(t *testing.T) {
	runner := New(Config{Gamma: &fakeGamma{}, Enricher: &fakeEnricher{}})

	got, err := runner.Run(context.Background(), Request{Operation: Categories})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	categories, ok := got.([]polytypes.PolymarketCategory)
	if !ok || len(categories) == 0 {
		t.Fatalf("result=%#v", got)
	}
	if categories[0].Slug != "trending" {
		t.Fatalf("first category=%+v", categories[0])
	}
}

func TestRunnerCategoryEventsUsesCuratedGammaFeed(t *testing.T) {
	gamma := &fakeGamma{}
	runner := New(Config{Gamma: gamma, Enricher: &fakeEnricher{}})

	got, err := runner.Run(context.Background(), Request{Operation: CategoryEvents, Slug: "mentions", Limit: 3, Order: "volume24hr"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	resp, ok := got.(*polytypes.CategoryEventsResponse)
	if !ok || len(resp.Events) != 1 || resp.NextCursor != "cursor-2" {
		t.Fatalf("result=%#v", got)
	}
	if gamma.categoryEventsCat.Slug != "mentions" || len(gamma.categoryEventsCat.TagSlugs) != 1 || gamma.categoryEventsCat.TagSlugs[0] != "tweets-markets" {
		t.Fatalf("category=%+v", gamma.categoryEventsCat)
	}
	if gamma.categoryEventsParam == nil || gamma.categoryEventsParam.Limit != 3 || gamma.categoryEventsParam.Order != "volume24hr" {
		t.Fatalf("params=%+v", gamma.categoryEventsParam)
	}
}

func TestRunnerEnrichErrorsWhenMarketNotFound(t *testing.T) {
	gamma := &fakeGamma{marketByIDNil: true}
	runner := New(Config{Gamma: gamma, Enricher: &fakeEnricher{}})

	_, err := runner.Run(context.Background(), Request{Operation: Enrich, ID: "missing"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v, want a 'not found' error (no panic)", err)
	}
}
