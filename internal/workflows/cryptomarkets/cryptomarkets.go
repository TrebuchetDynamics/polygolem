// Package cryptomarkets selects active crypto markets from Gamma search results.
package cryptomarkets

import (
	"encoding/json"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

// Filter contains crypto market search filters shared by workflow modules.
type Filter struct {
	Asset    string
	Interval string
}

// Candidate is one active Gamma market plus parsed CLOB token IDs.
type Candidate struct {
	Event    polytypes.Event
	Market   polytypes.Market
	TokenIDs []string
}

// Query builds the Gamma search query for a crypto market filter.
func Query(filter Filter) string {
	asset := strings.TrimSpace(filter.Asset)
	interval := strings.TrimSpace(filter.Interval)
	query := asset
	if interval != "" {
		if query != "" {
			query += " "
		}
		query += interval
	}
	if query == "" {
		query = "crypto"
	}
	return query
}

// Select returns active, non-closed event markets matching the filter.
func Select(resp *polytypes.SearchResponse, filter Filter) []Candidate {
	if resp == nil {
		return nil
	}
	var out []Candidate
	for _, event := range resp.Events {
		if !event.Active || event.Closed {
			continue
		}
		for _, market := range event.Markets {
			if !market.Active || market.Closed {
				continue
			}
			if !matchesAsset(event, market, filter.Asset) || !matchesInterval(event, market, filter.Interval) {
				continue
			}
			out = append(out, Candidate{Event: event, Market: market, TokenIDs: ParseTokenIDs(market.ClobTokenIDs)})
		}
	}
	return out
}

// ParseTokenIDs parses Gamma's CLOB token ID field, accepting either JSON arrays or raw IDs.
func ParseTokenIDs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return []string{raw}
	}
	return ids
}

func matchesAsset(event polytypes.Event, market polytypes.Market, asset string) bool {
	asset = strings.TrimSpace(asset)
	if asset == "" {
		return true
	}
	asset = strings.ToUpper(asset)
	return strings.Contains(strings.ToUpper(market.Question), asset) || strings.Contains(strings.ToUpper(event.Title), asset)
}

func matchesInterval(event polytypes.Event, market polytypes.Market, interval string) bool {
	interval = strings.TrimSpace(interval)
	if interval == "" {
		return true
	}
	interval = strings.ToLower(interval)
	return strings.Contains(strings.ToLower(event.Title), interval) || strings.Contains(strings.ToLower(market.Question), interval)
}
