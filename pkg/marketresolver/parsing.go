package marketresolver

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// Exported parsing helpers for crypto up/down market payloads. Consumers that
// read raw Gamma/CLOB market rows use these instead of re-implementing
// Polymarket's naming, slug, and outcome conventions.

// UpDownTokenIDs pairs a market's outcome labels with its CLOB token IDs and
// returns the Up and Down token IDs. Outcomes "up"/"yes" map to the up token,
// "down"/"no" to the down token (case- and whitespace-insensitive). Both
// results are empty when the slices have different lengths.
func UpDownTokenIDs(outcomes, tokenIDs []string) (up, down string) {
	if len(outcomes) != len(tokenIDs) {
		return "", ""
	}
	for i, o := range outcomes {
		switch strings.ToLower(strings.TrimSpace(o)) {
		case "up", "yes":
			up = tokenIDs[i]
		case "down", "no":
			down = tokenIDs[i]
		}
	}
	return up, down
}

// InferTimeframe detects a 5m or 15m crypto window timeframe from free text
// (slug, question, group title — pass any combination; they are joined). It
// recognizes the "5m"/"15m", "5 min"/"15 min", and "5-minute"/"15-minute"
// spellings Polymarket uses, checking 15m first so "15m" never matches as
// "5m". Returns "" when no timeframe is mentioned.
func InferTimeframe(text ...string) string {
	joined := strings.ToLower(strings.Join(text, " "))
	for _, tf := range []string{"15m", "15 min", "15-minute", "5m", "5 min", "5-minute"} {
		if strings.Contains(joined, tf) {
			if strings.HasPrefix(tf, "5") {
				return "5m"
			}
			return "15m"
		}
	}
	return ""
}

// InferTimeframeFromWindow classifies a market window's duration as "5m" or
// "15m" with tolerance for the ~1-minute listing skew Polymarket applies to
// window bounds. Returns "" for zero, negative, or longer windows.
func InferTimeframeFromWindow(start, end time.Time) string {
	if !start.Before(end) {
		return ""
	}
	d := end.Sub(start)
	switch {
	case d <= 6*time.Minute:
		return "5m"
	case d <= 16*time.Minute:
		return "15m"
	default:
		return ""
	}
}

// WindowFromSlug recovers the exact market window from a crypto up/down slug.
// Slugs end in the unix epoch seconds of the window start (see
// CryptoWindowSlug); the window end is start plus the timeframe duration.
// Returns ok=false when the slug carries no positive epoch suffix or the
// timeframe does not parse as a positive duration.
func WindowFromSlug(slug, timeframe string) (start, end time.Time, ok bool) {
	parts := strings.Split(strings.TrimSpace(slug), "-")
	if len(parts) == 0 {
		return time.Time{}, time.Time{}, false
	}
	epoch, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil || epoch <= 0 {
		return time.Time{}, time.Time{}, false
	}
	duration, err := time.ParseDuration(timeframe)
	if err != nil || duration <= 0 {
		return time.Time{}, time.Time{}, false
	}
	start = time.Unix(epoch, 0).UTC()
	return start, start.Add(duration), true
}

// AssetSearchQueries returns the Gamma search queries that find an asset's
// crypto up/down markets, covering the market-title spellings Polymarket
// uses (e.g. "bitcoin 5m", "dogecoin 15m"). Unknown assets fall back to the
// lowercased symbol.
func AssetSearchQueries(asset string) []string {
	names := assetSearchNames(asset)
	queries := make([]string, 0, len(names)*2)
	for _, name := range names {
		queries = append(queries, name+" 5m", name+" 15m")
	}
	return queries
}

// AssetMentioned reports whether free text mentions the asset by symbol or by
// any of the market-title spellings AssetSearchQueries covers.
func AssetMentioned(asset, text string) bool {
	text = strings.ToLower(text)
	if strings.Contains(text, strings.ToLower(strings.TrimSpace(asset))) {
		return true
	}
	for _, name := range assetSearchNames(asset) {
		if strings.Contains(text, name) {
			return true
		}
	}
	return false
}

func assetSearchNames(asset string) []string {
	names := map[string][]string{
		"BTC":  {"bitcoin"},
		"ETH":  {"ethereum"},
		"SOL":  {"solana"},
		"XRP":  {"xrp"},
		"DOGE": {"dogecoin", "doge"},
		"BNB":  {"bnb", "binance coin"},
	}[strings.ToUpper(strings.TrimSpace(asset))]
	if len(names) == 0 {
		names = []string{strings.ToLower(strings.TrimSpace(asset))}
	}
	return names
}

// ParseJSONStringList decodes the JSON-encoded string arrays Gamma embeds as
// strings (outcomes, clobTokenIds). Blank input decodes to nil.
func ParseJSONStringList(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	return values, nil
}
