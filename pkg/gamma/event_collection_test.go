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

func TestActiveEventsAllCollectsPagesAndDedupes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events" {
			t.Fatalf("path = %q, want /events", r.URL.Path)
		}
		if got := r.URL.Query().Get("closed"); got != "false" {
			t.Fatalf("closed = %q, want false", got)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Fatalf("limit = %q, want 100", got)
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var page []types.Event
		switch offset {
		case 0:
			for i := 0; i < 100; i++ {
				page = append(page, types.Event{ID: "e" + strconv.Itoa(i), Slug: "event-" + strconv.Itoa(i), Title: "Event", Active: true, Closed: false})
			}
		case 100:
			page = []types.Event{
				{ID: "duplicate", Slug: "event-99", Title: "Duplicate", Active: true, Closed: false},
				{ID: "e100", Slug: "event-100", Title: "Page two", Active: true, Closed: false},
			}
		default:
			page = []types.Event{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(page)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	events, err := client.ActiveEventsAll(context.Background())
	if err != nil {
		t.Fatalf("ActiveEventsAll returned error: %v", err)
	}
	if got, want := len(events), 101; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
	if events[len(events)-1].Slug != "event-100" {
		t.Fatalf("last slug = %q, want event-100", events[len(events)-1].Slug)
	}
}

func TestFilterEventsByCategoryKeepsTechSeparateFromScience(t *testing.T) {
	events := []types.Event{
		{ID: "science", Slug: "science-event", Category: "Science"},
		{ID: "technology", Slug: "technology-event", Category: "Technology"},
	}

	tech := FilterEventsByCategory(events, "Tech")

	if len(tech) != 1 || tech[0].ID != "technology" {
		t.Fatalf("Tech filter = %#v", tech)
	}
}

func TestFilterEventsByCategoryKeepsWorldSeparateFromPolitics(t *testing.T) {
	events := []types.Event{
		{ID: "politics", Slug: "politics-event", Category: "Politics"},
		{ID: "world", Slug: "world-event", Category: "World"},
	}

	world := FilterEventsByCategory(events, "World")

	if len(world) != 1 || world[0].ID != "world" {
		t.Fatalf("World filter = %#v", world)
	}
}

func TestFilterEventsByCategoryKeepsWeatherSeparateFromScience(t *testing.T) {
	events := []types.Event{
		{ID: "science", Slug: "science-event", Category: "Science"},
		{ID: "weather", Slug: "weather-event", Category: "Weather"},
	}

	weather := FilterEventsByCategory(events, "Weather")

	if len(weather) != 1 || weather[0].ID != "weather" {
		t.Fatalf("Weather filter = %#v", weather)
	}
}

func TestFilterEventsByCategoryAliases(t *testing.T) {
	events := []types.Event{
		{ID: "business", Slug: "business-event", Category: "Business"},
		{ID: "technology", Slug: "technology-event", Category: "Technology"},
		{ID: "politics", Slug: "politics-event", Category: "Politics"},
	}

	finance := FilterEventsByCategory(events, "Finance")
	tech := FilterEventsByCategory(events, "Tech")
	elections := FilterEventsByCategory(events, "Elections")

	if len(finance) != 1 || finance[0].ID != "business" {
		t.Fatalf("Finance filter = %#v", finance)
	}
	if len(tech) != 1 || tech[0].ID != "technology" {
		t.Fatalf("Tech filter = %#v", tech)
	}
	if len(elections) != 1 || elections[0].ID != "politics" {
		t.Fatalf("Elections filter = %#v", elections)
	}
}
