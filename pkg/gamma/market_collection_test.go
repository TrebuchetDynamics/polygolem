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

func TestActiveMarketsAllCollectsPagesAndDedupes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/markets" {
			t.Fatalf("path = %q, want /markets", r.URL.Path)
		}
		if got := r.URL.Query().Get("active"); got != "true" {
			t.Fatalf("active = %q, want true", got)
		}
		if got := r.URL.Query().Get("closed"); got != "false" {
			t.Fatalf("closed = %q, want false", got)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Fatalf("limit = %q, want 100", got)
		}
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var page []types.Market
		switch offset {
		case 0:
			for i := 0; i < 100; i++ {
				page = append(page, types.Market{ConditionID: "c" + strconv.Itoa(i), Question: "Market", Active: true, Closed: false})
			}
		case 100:
			page = []types.Market{
				{ConditionID: "c99", Question: "Duplicate", Active: true, Closed: false},
				{ConditionID: "c100", Question: "Page two", Active: true, Closed: false},
			}
		default:
			page = []types.Market{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(page)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	markets, err := client.ActiveMarketsAll(context.Background())
	if err != nil {
		t.Fatalf("ActiveMarketsAll returned error: %v", err)
	}
	if got, want := len(markets), 101; got != want {
		t.Fatalf("len(markets) = %d, want %d", got, want)
	}
	if markets[len(markets)-1].ConditionID != "c100" {
		t.Fatalf("last condition = %q, want c100", markets[len(markets)-1].ConditionID)
	}
}

func TestFilterMarketsByCategoryAliases(t *testing.T) {
	markets := []types.Market{
		{ConditionID: "business", Category: "Business"},
		{ConditionID: "science", Category: "Science"},
		{ConditionID: "politics", Category: "Politics"},
	}

	finance := FilterMarketsByCategory(markets, "Finance")
	tech := FilterMarketsByCategory(markets, "Tech")
	elections := FilterMarketsByCategory(markets, "Elections")

	if len(finance) != 1 || finance[0].ConditionID != "business" {
		t.Fatalf("Finance filter = %#v", finance)
	}
	if len(tech) != 1 || tech[0].ConditionID != "science" {
		t.Fatalf("Tech filter = %#v", tech)
	}
	if len(elections) != 1 || elections[0].ConditionID != "politics" {
		t.Fatalf("Elections filter = %#v", elections)
	}
}
