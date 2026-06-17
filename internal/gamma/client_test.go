package gamma

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/transport"
)

func TestActiveMarketsUsesContextAndParsesMarkets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/markets" {
			t.Fatalf("path = %q, want /markets", r.URL.Path)
		}
		if r.URL.Query().Get("active") != "true" {
			t.Fatalf("active query = %q, want true", r.URL.Query().Get("active"))
		}
		if r.URL.Query().Get("closed") != "false" {
			t.Fatalf("closed query = %q, want false", r.URL.Query().Get("closed"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"1","slug":"m-1","question":"Will it rain?","active":true,"closed":false}]`))
	}))
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc)
	markets, err := client.ActiveMarkets(context.Background())
	if err != nil {
		t.Fatalf("ActiveMarkets returned error: %v", err)
	}
	if len(markets) != 1 || markets[0].Slug != "m-1" {
		t.Fatalf("unexpected markets: %#v", markets)
	}
}

func TestMarketsWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "5" {
			t.Fatalf("limit = %q, want 5", r.URL.Query().Get("limit"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc)

	active := true
	_, err := client.Markets(context.Background(), &polytypes.GetMarketsParams{
		Limit:  5,
		Active: &active,
	})
	if err != nil {
		t.Fatalf("Markets returned error: %v", err)
	}
}

func TestHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":"ok"}`))
	}))
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc)

	resp, err := client.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}
	if resp.Data != "ok" {
		t.Fatalf("HealthCheck Data = %q, want ok", resp.Data)
	}
}

func TestSearchCallsCorrectEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public-search" {
			t.Fatalf("path = %q, want /public-search", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "btc" {
			t.Fatalf("q = %q, want btc", r.URL.Query().Get("q"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"events":[],"tags":[],"profiles":[],"pagination":{"hasMore":false,"totalResults":0}}`))
	}))
	defer server.Close()

	cfg := transport.DefaultConfig(server.URL + "/")
	cfg.RetryMax = 0
	tc := transport.New(server.Client(), cfg)
	client := NewClient(server.URL+"/", tc)

	resp, err := client.Search(context.Background(), &polytypes.SearchParams{
		Q: "btc",
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if resp.Pagination.TotalResults != 0 {
		t.Fatalf("unexpected pagination: %#v", resp.Pagination)
	}
}

func TestEventBySlugUsesEventsSlugQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events" {
			t.Fatalf("path = %q, want /events", r.URL.Path)
		}
		if r.URL.Query().Get("slug") != "btc-updown-5m-1778115300" {
			t.Fatalf("slug query = %q", r.URL.Query().Get("slug"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{
			"id":"event-1",
			"slug":"btc-updown-5m-1778115300",
			"title":"Bitcoin Up or Down - May 6, 8:55PM-9:00PM ET",
			"markets":[{"id":"market-1","slug":"btc-updown-5m-1778115300"}]
		}]`))
	}))
	defer server.Close()

	cfg := transport.DefaultConfig(server.URL + "/")
	cfg.RetryMax = 0
	tc := transport.New(server.Client(), cfg)
	client := NewClient(server.URL+"/", tc)

	event, err := client.EventBySlug(context.Background(), "btc-updown-5m-1778115300")
	if err != nil {
		t.Fatalf("EventBySlug returned error: %v", err)
	}
	if event.ID != "event-1" || len(event.Markets) != 1 {
		t.Fatalf("unexpected event: %#v", event)
	}
}

func TestCommentsUsesCurrentParentEntityQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/comments" {
			t.Fatalf("path = %q, want /comments", r.URL.Path)
		}
		if got := r.URL.Query().Get("parent_entity_id"); got != "2144505" {
			t.Fatalf("parent_entity_id = %q, want 2144505", got)
		}
		if got := r.URL.Query().Get("parent_entity_type"); got != "Event" {
			t.Fatalf("parent_entity_type = %q, want Event", got)
		}
		if stale := r.URL.Query().Get("entity_id"); stale != "" {
			t.Fatalf("stale entity_id query was sent: %q", stale)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	cfg := transport.DefaultConfig(server.URL + "/")
	cfg.RetryMax = 0
	tc := transport.New(server.Client(), cfg)
	client := NewClient(server.URL+"/", tc)

	entityID := 2144505
	entityType := "Event"
	if _, err := client.Comments(context.Background(), &polytypes.CommentQuery{
		EntityID:   &entityID,
		EntityType: &entityType,
		Limit:      3,
	}); err != nil {
		t.Fatalf("Comments returned error: %v", err)
	}
}

// TestMarketsSerializesPreviouslyDroppedFilters guards the regression where
// buildQueryPath silently omitted several declared GetMarketsParams filters,
// returning unfiltered results without error.
func TestMarketsSerializesPreviouslyDroppedFilters(t *testing.T) {
	var got url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc)

	related := true
	rewards := 12.5
	startMin := polytypes.NormalizedTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	_, err := client.Markets(context.Background(), &polytypes.GetMarketsParams{
		ID:             []int{101, 102},
		RelatedTags:    &related,
		RewardsMinSize: &rewards,
		GameID:         "game-7",
		StartDateMin:   &startMin,
	})
	if err != nil {
		t.Fatalf("Markets returned error: %v", err)
	}
	if ids := got["id"]; len(ids) != 2 || ids[0] != "101" || ids[1] != "102" {
		t.Fatalf("id = %v, want [101 102]", ids)
	}
	if got.Get("related_tags") != "true" {
		t.Fatalf("related_tags = %q, want true", got.Get("related_tags"))
	}
	if got.Get("rewards_min_size") != "12.5" {
		t.Fatalf("rewards_min_size = %q, want 12.5", got.Get("rewards_min_size"))
	}
	if got.Get("game_id") != "game-7" {
		t.Fatalf("game_id = %q, want game-7", got.Get("game_id"))
	}
	if got.Get("start_date_min") != "2026-01-02T03:04:05Z" {
		t.Fatalf("start_date_min = %q, want 2026-01-02T03:04:05Z", got.Get("start_date_min"))
	}
}

// TestEventsSerializesPreviouslyDroppedFilters guards the same regression for
// GetEventsParams (featured, recurrence, and date filters were dropped).
func TestEventsSerializesPreviouslyDroppedFilters(t *testing.T) {
	var got url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc)

	featured := true
	endMax := polytypes.NormalizedTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	_, err := client.Events(context.Background(), &polytypes.GetEventsParams{
		ID:         []int{5},
		Featured:   &featured,
		Recurrence: "weekly",
		EndDateMax: &endMax,
	})
	if err != nil {
		t.Fatalf("Events returned error: %v", err)
	}
	if got.Get("featured") != "true" {
		t.Fatalf("featured = %q, want true", got.Get("featured"))
	}
	if got.Get("recurrence") != "weekly" {
		t.Fatalf("recurrence = %q, want weekly", got.Get("recurrence"))
	}
	if ids := got["id"]; len(ids) != 1 || ids[0] != "5" {
		t.Fatalf("id = %v, want [5]", ids)
	}
	if got.Get("end_date_max") != "2026-06-01T00:00:00Z" {
		t.Fatalf("end_date_max = %q, want 2026-06-01T00:00:00Z", got.Get("end_date_max"))
	}
}

// TestCommentsByUserURLEncodesAddress guards the regression where the user
// address was interpolated into the query string without URL-encoding, so a
// value containing query-significant characters could break or inject params.
func TestCommentsByUserURLEncodesAddress(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	tc := transport.New(server.Client(), transport.DefaultConfig(server.URL+"/"))
	client := NewClient(server.URL+"/", tc)

	addr := "0xAbC 123&injected=1"
	if _, err := client.CommentsByUser(context.Background(), addr, 25); err != nil {
		t.Fatalf("CommentsByUser returned error: %v", err)
	}
	if gotQuery.Get("user_address") != addr {
		t.Fatalf("user_address = %q, want %q (round-trips through proper encoding)", gotQuery.Get("user_address"), addr)
	}
	if gotQuery.Has("injected") {
		t.Fatal("query-significant characters in the address leaked an injected parameter")
	}
	if gotQuery.Get("limit") != "25" {
		t.Fatalf("limit = %q, want 25", gotQuery.Get("limit"))
	}
}
