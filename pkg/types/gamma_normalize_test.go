package types

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNormalizedTimeParsesSecondsGranularityOffset guards the regression where
// Gamma returned a timestamp with a seconds-granularity timezone offset
// (e.g. "...+00:00:00") that no layout matched, breaking response decoding.
func TestNormalizedTimeParsesSecondsGranularityOffset(t *testing.T) {
	cases := map[string]time.Time{
		`"2026-06-16 22:35:33.941001+00:00:00"`: time.Date(2026, 6, 16, 22, 35, 33, 941001000, time.UTC),
		`"2026-06-16T22:35:33-05:00:00"`:        time.Date(2026, 6, 16, 22, 35, 33, 0, time.FixedZone("", -5*3600)),
		// Existing well-formed inputs must still parse.
		`"2026-06-16T22:35:33Z"`:      time.Date(2026, 6, 16, 22, 35, 33, 0, time.UTC),
		`"2026-06-16 22:35:33+00:00"`: time.Date(2026, 6, 16, 22, 35, 33, 0, time.UTC),
		`"2026-06-16"`:                time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC),
	}
	for raw, want := range cases {
		var nt NormalizedTime
		if err := json.Unmarshal([]byte(raw), &nt); err != nil {
			t.Fatalf("%s: unexpected error %v", raw, err)
		}
		if !nt.Time().Equal(want) {
			t.Fatalf("%s: parsed %s, want %s", raw, nt.Time().UTC(), want.UTC())
		}
	}
}
