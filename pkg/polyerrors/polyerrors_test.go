package polyerrors

import "testing"

func TestNormalizeClassifiesHTTPStatus(t *testing.T) {
	cases := []struct {
		name   string
		status int
		want   Kind
	}{
		{"rate limited", 429, RateLimited},
		{"unauthorized", 401, AuthRejected},
		{"geoblock", 403, Geoblocked},
		{"server", 502, UpstreamUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Normalize(Input{HTTPStatus: tc.status})
			if got.Kind != tc.want {
				t.Fatalf("kind=%s want %s", got.Kind, tc.want)
			}
		})
	}
}

func TestNormalizeClassifiesPolymarketMessages(t *testing.T) {
	cases := []struct {
		message string
		want    Kind
	}{
		{"order rejected: invalid tick size", TickSizeMismatch},
		{"market is closed", MarketClosed},
		{"not enough balance / insufficient funds", InsufficientFunds},
		{"user is blocked by geoblock", Geoblocked},
		{"API key expired", AuthRejected},
	}
	for _, tc := range cases {
		got := Normalize(Input{Message: tc.message})
		if got.Kind != tc.want {
			t.Fatalf("%q kind=%s want %s", tc.message, got.Kind, tc.want)
		}
	}
}

func TestNormalizeRedactsSecretBearingHeaders(t *testing.T) {
	got := Normalize(Input{
		Source:  "clob",
		Message: "POLY_API_KEY=abc POLY_SIGNATURE=def POLY_PASSPHRASE=ghi private_key=0x123 bearer token",
	})
	if got.Message != "[redacted]" {
		t.Fatalf("message=%q", got.Message)
	}
	if got.Source != "clob" {
		t.Fatalf("source=%q", got.Source)
	}
}

func TestNormalizeUnknownKeepsSafeMessage(t *testing.T) {
	got := Normalize(Input{Message: "weird upstream error"})
	if got.Kind != Unknown {
		t.Fatalf("kind=%s", got.Kind)
	}
	if got.Message != "weird upstream error" {
		t.Fatalf("message=%q", got.Message)
	}
}
