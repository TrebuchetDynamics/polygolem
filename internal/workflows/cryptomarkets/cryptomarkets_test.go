package cryptomarkets

import (
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

func TestQueryBuildsAssetIntervalAndDefaultsToCrypto(t *testing.T) {
	for _, tc := range []struct {
		name   string
		filter Filter
		want   string
	}{
		{name: "asset interval", filter: Filter{Asset: "BTC", Interval: "5m"}, want: "bitcoin 5m updown"},
		{name: "trimmed asset interval", filter: Filter{Asset: " ETH ", Interval: " 15m "}, want: "ethereum 15m updown"},
		{name: "asset only", filter: Filter{Asset: "SOL"}, want: "SOL"},
		{name: "interval only", filter: Filter{Interval: "daily"}, want: "daily updown"},
		{name: "empty", filter: Filter{}, want: "crypto"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Query(tc.filter); got != tc.want {
				t.Fatalf("Query(%+v)=%q, want %q", tc.filter, got, tc.want)
			}
		})
	}
}

func TestSelectFiltersActiveMarketsByAssetAndIntervalAndParsesTokens(t *testing.T) {
	resp := &polytypes.SearchResponse{Events: []polytypes.Event{
		{
			ID:     "event-btc",
			Title:  "BTC 5m event",
			Active: true,
			Markets: []polytypes.Market{
				{
					ID:           "market-btc-json",
					Question:     "Will BTC go up in 5m?",
					Active:       true,
					ClobTokenIDs: `["btc-up","btc-down"]`,
				},
				{
					ID:           "market-btc-raw",
					Question:     "BTC raw 5m market",
					Active:       true,
					ClobTokenIDs: `btc-raw-token`,
				},
				{
					ID:           "market-closed",
					Question:     "BTC closed 5m market",
					Active:       true,
					Closed:       true,
					ClobTokenIDs: `["closed"]`,
				},
			},
		},
		{
			ID:     "event-eth",
			Title:  "ETH 5m event",
			Active: true,
			Markets: []polytypes.Market{{
				ID:           "market-eth",
				Question:     "ETH up in 5m?",
				Active:       true,
				ClobTokenIDs: `["eth-up"]`,
			}},
		},
		{
			ID:     "event-closed",
			Title:  "BTC 5m closed event",
			Active: true,
			Closed: true,
			Markets: []polytypes.Market{{
				ID:           "hidden",
				Question:     "BTC hidden 5m?",
				Active:       true,
				ClobTokenIDs: `["hidden"]`,
			}},
		},
		{
			ID:     "event-inactive",
			Title:  "BTC 5m inactive event",
			Active: false,
			Markets: []polytypes.Market{{
				ID:           "inactive",
				Question:     "BTC inactive 5m?",
				Active:       true,
				ClobTokenIDs: `["inactive"]`,
			}},
		},
	}}

	got := Select(resp, Filter{Asset: "btc", Interval: "5M"})
	if len(got) != 2 {
		t.Fatalf("Select returned %d candidates, want 2: %+v", len(got), got)
	}
	if got[0].Event.ID != "event-btc" || got[0].Market.ID != "market-btc-json" {
		t.Fatalf("unexpected first candidate: %+v", got[0])
	}
	if want := []string{"btc-up", "btc-down"}; !reflect.DeepEqual(got[0].TokenIDs, want) {
		t.Fatalf("first tokens=%v, want %v", got[0].TokenIDs, want)
	}
	if got[1].Market.ID != "market-btc-raw" {
		t.Fatalf("unexpected second candidate: %+v", got[1])
	}
	if want := []string{"btc-raw-token"}; !reflect.DeepEqual(got[1].TokenIDs, want) {
		t.Fatalf("second tokens=%v, want %v", got[1].TokenIDs, want)
	}
}

func TestSelectMatchesIntervalInSlug(t *testing.T) {
	resp := &polytypes.SearchResponse{Events: []polytypes.Event{{
		ID:     "event-btc",
		Title:  "Bitcoin Up or Down - June 27, 11:50PM-11:55PM ET",
		Slug:   "btc-updown-5m-1782618600",
		Active: true,
		Markets: []polytypes.Market{{
			ID:           "market-btc",
			Slug:         "btc-updown-5m-1782618600",
			Question:     "Bitcoin Up or Down - June 27, 11:50PM-11:55PM ET",
			Active:       true,
			ClobTokenIDs: `["up","down"]`,
		}},
	}}}

	got := Select(resp, Filter{Asset: "BTC", Interval: "5m"})
	if len(got) != 1 {
		t.Fatalf("Select returned %d candidates, want 1", len(got))
	}
}

func TestSelectHandlesNilResponseAndEmptyTokenIDs(t *testing.T) {
	if got := Select(nil, Filter{}); got != nil {
		t.Fatalf("Select(nil)=%v, want nil", got)
	}
	resp := &polytypes.SearchResponse{Events: []polytypes.Event{{
		Title:  "BTC 5m event",
		Active: true,
		Markets: []polytypes.Market{{
			ID:           "market-btc",
			Question:     "BTC up in 5m?",
			Active:       true,
			ClobTokenIDs: "",
		}},
	}}}
	got := Select(resp, Filter{Asset: "BTC", Interval: "5m"})
	if len(got) != 1 {
		t.Fatalf("Select returned %d candidates, want 1", len(got))
	}
	if got[0].TokenIDs != nil {
		t.Fatalf("tokens=%v, want nil", got[0].TokenIDs)
	}
}
