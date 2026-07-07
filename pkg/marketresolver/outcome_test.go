package marketresolver

import "testing"

func TestNormalizeOutcome(t *testing.T) {
	cases := map[string]string{
		"up":      OutcomeUp,
		" UP ":    OutcomeUp,
		"Down":    OutcomeDown,
		"yes":     OutcomeUnknown,
		"":        OutcomeUnknown,
		"unknown": OutcomeUnknown,
	}
	for in, want := range cases {
		if got := NormalizeOutcome(in); got != want {
			t.Fatalf("NormalizeOutcome(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestOutcomeForToken(t *testing.T) {
	const up, down = "111", "222"
	cases := []struct {
		name    string
		winning string
		want    string
	}{
		{name: "up token wins", winning: "111", want: OutcomeUp},
		{name: "down token wins", winning: "222", want: OutcomeDown},
		{name: "whitespace trimmed", winning: " 111 ", want: OutcomeUp},
		{name: "empty winning token", winning: "", want: OutcomeUnknown},
		{name: "foreign token", winning: "999", want: OutcomeUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := OutcomeForToken(tc.winning, up, down); got != tc.want {
				t.Fatalf("OutcomeForToken(%q) = %q; want %q", tc.winning, got, tc.want)
			}
		})
	}
}

func TestOutcomeForTokenEmptyPairDoesNotMatchEmptyWinner(t *testing.T) {
	if got := OutcomeForToken("", "", ""); got != OutcomeUnknown {
		t.Fatalf("empty winner should be unknown, got %q", got)
	}
	if got := OutcomeForToken("111", "", ""); got != OutcomeUnknown {
		t.Fatalf("winner against empty pair should be unknown, got %q", got)
	}
}

func TestCryptoMarketOutcomeForToken(t *testing.T) {
	m := CryptoMarket{UpTokenID: "111", DownTokenID: "222"}
	if got := m.OutcomeForToken("222"); got != OutcomeDown {
		t.Fatalf("market OutcomeForToken = %q; want down", got)
	}
}
