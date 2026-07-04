package marketresolver

import (
	"testing"
	"time"
)

func TestUpDownTokenIDs(t *testing.T) {
	up, down := UpDownTokenIDs([]string{"Up", "Down"}, []string{"111", "222"})
	if up != "111" || down != "222" {
		t.Fatalf("up/down = %q/%q", up, down)
	}
	up, down = UpDownTokenIDs([]string{" yes ", " NO "}, []string{"111", "222"})
	if up != "111" || down != "222" {
		t.Fatalf("yes/no mapping = %q/%q", up, down)
	}
	if up, down = UpDownTokenIDs([]string{"Up"}, []string{"1", "2"}); up != "" || down != "" {
		t.Fatal("length mismatch should return empty pair")
	}
	if up, down = UpDownTokenIDs([]string{"Over", "Under"}, []string{"1", "2"}); up != "" || down != "" {
		t.Fatal("foreign outcomes should return empty pair")
	}
}

func TestInferTimeframe(t *testing.T) {
	cases := []struct {
		text []string
		want string
	}{
		{[]string{"btc-updown-5m-1751500800"}, "5m"},
		{[]string{"Bitcoin Up or Down - 15 min"}, "15m"},
		{[]string{"eth 15-minute window"}, "15m"},
		{[]string{"slug-without-window", "Will ETH close higher?"}, ""},
		{[]string{"btc", "5 min sprint"}, "5m"},
		// 15m must win over the "5m" substring it contains.
		{[]string{"doge-updown-15m-1751500800"}, "15m"},
	}
	for _, tc := range cases {
		if got := InferTimeframe(tc.text...); got != tc.want {
			t.Fatalf("InferTimeframe(%v) = %q; want %q", tc.text, got, tc.want)
		}
	}
}

func TestInferTimeframeFromWindow(t *testing.T) {
	base := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		minutes int
		want    string
	}{
		{5, "5m"}, {6, "5m"}, {15, "15m"}, {16, "15m"}, {17, ""}, {0, ""}, {-5, ""},
	}
	for _, tc := range cases {
		got := InferTimeframeFromWindow(base, base.Add(time.Duration(tc.minutes)*time.Minute))
		if got != tc.want {
			t.Fatalf("window %dmin = %q; want %q", tc.minutes, got, tc.want)
		}
	}
}

func TestWindowFromSlugRoundTripsCryptoWindowSlug(t *testing.T) {
	windowStart := time.Date(2026, 7, 3, 12, 5, 0, 0, time.UTC)
	slug := CryptoWindowSlug("BTC", "5m", windowStart)
	start, end, ok := WindowFromSlug(slug, "5m")
	if !ok {
		t.Fatalf("WindowFromSlug(%q) not ok", slug)
	}
	if !start.Equal(windowStart) || !end.Equal(windowStart.Add(5*time.Minute)) {
		t.Fatalf("window = %s..%s", start, end)
	}
}

func TestWindowFromSlugRejectsBadInput(t *testing.T) {
	if _, _, ok := WindowFromSlug("btc-updown-5m-notanumber", "5m"); ok {
		t.Fatal("non-numeric epoch accepted")
	}
	if _, _, ok := WindowFromSlug("btc-updown-5m-0", "5m"); ok {
		t.Fatal("zero epoch accepted")
	}
	if _, _, ok := WindowFromSlug("btc-updown-5m-1751500800", "bogus"); ok {
		t.Fatal("unparseable timeframe accepted")
	}
	if _, _, ok := WindowFromSlug("", "5m"); ok {
		t.Fatal("empty slug accepted")
	}
}

func TestAssetSearchQueries(t *testing.T) {
	got := AssetSearchQueries("DOGE")
	want := []string{"dogecoin 5m", "dogecoin 15m", "doge 5m", "doge 15m"}
	if len(got) != len(want) {
		t.Fatalf("queries = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("queries[%d] = %q; want %q", i, got[i], want[i])
		}
	}
	fallback := AssetSearchQueries("hype")
	if len(fallback) != 2 || fallback[0] != "hype 5m" {
		t.Fatalf("fallback queries = %v", fallback)
	}
}

func TestAssetMentioned(t *testing.T) {
	if !AssetMentioned("BTC", "Bitcoin Up or Down - Jul 3") {
		t.Fatal("bitcoin spelling not recognized for BTC")
	}
	if !AssetMentioned("BNB", "binance coin 5m window") {
		t.Fatal("binance coin spelling not recognized for BNB")
	}
	if !AssetMentioned("ETH", "eth-updown-5m-1751500800") {
		t.Fatal("symbol mention not recognized")
	}
	if AssetMentioned("SOL", "Bitcoin Up or Down") {
		t.Fatal("SOL claimed mention in a bitcoin title")
	}
}

func TestParseJSONStringList(t *testing.T) {
	got, err := ParseJSONStringList(`["Up","Down"]`)
	if err != nil || len(got) != 2 || got[0] != "Up" {
		t.Fatalf("ParseJSONStringList = %v, %v", got, err)
	}
	got, err = ParseJSONStringList("  ")
	if err != nil || got != nil {
		t.Fatalf("blank input = %v, %v; want nil, nil", got, err)
	}
	if _, err := ParseJSONStringList(`{"not":"a list"}`); err == nil {
		t.Fatal("object accepted as string list")
	}
}
