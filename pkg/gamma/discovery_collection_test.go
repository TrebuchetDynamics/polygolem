package gamma

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

func TestActiveSeriesAllCollectsPagesAndDedupes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/series" {
			t.Fatalf("path = %q, want /series", r.URL.Path)
		}
		if got := r.URL.Query().Get("closed"); got != "false" {
			t.Fatalf("closed = %q, want false", got)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Fatalf("limit = %q, want 100", got)
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var page []types.Series
		switch offset {
		case 0:
			for i := 0; i < 100; i++ {
				page = append(page, types.Series{ID: "s" + strconv.Itoa(i), Slug: "series-" + strconv.Itoa(i), Title: "Series", Active: true, Closed: false})
			}
		case 100:
			page = []types.Series{
				{ID: "duplicate", Slug: "series-99", Title: "Duplicate", Active: true, Closed: false},
				{ID: "s100", Slug: "series-100", Title: "Page two", Active: true, Closed: false},
			}
		default:
			page = []types.Series{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(page)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	series, err := client.ActiveSeriesAll(context.Background())
	if err != nil {
		t.Fatalf("ActiveSeriesAll returned error: %v", err)
	}
	if got, want := len(series), 101; got != want {
		t.Fatalf("len(series) = %d, want %d", got, want)
	}
	if series[len(series)-1].Slug != "series-100" {
		t.Fatalf("last slug = %q, want series-100", series[len(series)-1].Slug)
	}
}

func TestTagsAllCollectsPagesAndDedupes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tags" {
			t.Fatalf("path = %q, want /tags", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Fatalf("limit = %q, want 100", got)
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var page []types.Tag
		switch offset {
		case 0:
			for i := 0; i < 100; i++ {
				page = append(page, types.Tag{ID: strconv.Itoa(i), Slug: "tag-" + strconv.Itoa(i), Label: "Tag"})
			}
		case 100:
			page = []types.Tag{
				{ID: "duplicate", Slug: "tag-99", Label: "Duplicate"},
				{ID: "100", Slug: "tag-100", Label: "Page two"},
			}
		default:
			page = []types.Tag{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(page)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	tags, err := client.TagsAll(context.Background())
	if err != nil {
		t.Fatalf("TagsAll returned error: %v", err)
	}
	if got, want := len(tags), 101; got != want {
		t.Fatalf("len(tags) = %d, want %d", got, want)
	}
	if tags[len(tags)-1].Slug != "tag-100" {
		t.Fatalf("last slug = %q, want tag-100", tags[len(tags)-1].Slug)
	}
}
