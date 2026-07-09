package gamma

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

func TestPolymarketCategoriesIncludesCuratedNavigation(t *testing.T) {
	categories := PolymarketCategories()
	want := map[string]string{
		"politics":  "Politics",
		"world-cup": "World Cup",
		"breaking":  "Breaking",
		"mentions":  "Mentions",
		"all":       "All",
	}
	for slug, label := range want {
		cat, ok := PolymarketCategoryBySlug(slug)
		if !ok {
			t.Fatalf("category %q missing from %#v", slug, categories)
		}
		if cat.Label != label {
			t.Fatalf("category %q label=%q want %q", slug, cat.Label, label)
		}
	}
	mentions, _ := PolymarketCategoryBySlug("mentions")
	if len(mentions.TagSlugs) != 1 || mentions.TagSlugs[0] != "tweets-markets" {
		t.Fatalf("mentions tag slugs=%v", mentions.TagSlugs)
	}
	worldCup, _ := PolymarketCategoryBySlug("world-cup")
	if len(worldCup.TagSlugs) == 0 || worldCup.TagSlugs[0] != "fifa-world-cup" {
		t.Fatalf("world cup tag slugs=%v", worldCup.TagSlugs)
	}
	breaking, _ := PolymarketCategoryBySlug("breaking")
	if breaking.FeedMode != "route_only" {
		t.Fatalf("breaking feed mode=%q", breaking.FeedMode)
	}
}

func TestCategoryEventsUsesGammaEventsKeysetTagSlug(t *testing.T) {
	var gotPath, gotTagSlug, gotOrder, gotClosed, gotLimit string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		q := r.URL.Query()
		gotTagSlug = q.Get("tag_slug")
		gotOrder = q.Get("order")
		gotClosed = q.Get("closed")
		gotLimit = q.Get("limit")
		_ = json.NewEncoder(w).Encode(struct {
			Events     []types.Event `json:"events"`
			NextCursor string        `json:"next_cursor"`
		}{Events: []types.Event{{ID: "event-1", Slug: "event-one", Title: "Event One"}}, NextCursor: "cursor-2"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	closed := false
	resp, err := client.CategoryEvents(context.Background(), "politics", &types.CategoryEventsParams{Limit: 7, Order: "volume24hr", Closed: &closed})
	if err != nil {
		t.Fatalf("CategoryEvents returned error: %v", err)
	}
	if gotPath != "/events/keyset" || gotTagSlug != "politics" || gotOrder != "volume24hr" || gotClosed != "false" || gotLimit != "7" {
		t.Fatalf("request path=%q tag=%q order=%q closed=%q limit=%q", gotPath, gotTagSlug, gotOrder, gotClosed, gotLimit)
	}
	if resp.Category.Slug != "politics" || len(resp.Events) != 1 || resp.NextCursor != "cursor-2" || !resp.HasMore {
		t.Fatalf("response=%+v", resp)
	}
}

func TestCategoryEventsAllUsesGammaEventsKeysetWithoutTagSlug(t *testing.T) {
	var gotTagSlug string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events/keyset" {
			t.Fatalf("path = %q, want /events/keyset", r.URL.Path)
		}
		gotTagSlug = r.URL.Query().Get("tag_slug")
		_ = json.NewEncoder(w).Encode(struct {
			Events []types.Event `json:"events"`
		}{Events: []types.Event{{ID: "event-1", Slug: "event-one"}}})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.CategoryEvents(context.Background(), "all", &types.CategoryEventsParams{Limit: 1})
	if err != nil {
		t.Fatalf("CategoryEvents returned error: %v", err)
	}
	if gotTagSlug != "" {
		t.Fatalf("tag_slug = %q, want empty for all feed", gotTagSlug)
	}
	if resp.Category.Slug != "all" || len(resp.Events) != 1 {
		t.Fatalf("response=%+v", resp)
	}
}

func TestCategoryEventsRejectsRouteOnlyCategory(t *testing.T) {
	client := NewClient("http://127.0.0.1")
	_, err := client.CategoryEvents(context.Background(), "breaking", nil)
	if err == nil {
		t.Fatal("CategoryEvents returned nil error for route-only category")
	}
}
