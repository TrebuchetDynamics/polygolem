package upstreamdrift

import "testing"

func TestCheckLLMSPassesWhenKnownOfficialSectionsExist(t *testing.T) {
	text := `
https://docs.polymarket.com/market-data/overview
https://docs.polymarket.com/trading/overview
https://docs.polymarket.com/api-reference/core/get-current-positions-for-a-user
https://docs.polymarket.com/api-reference/relayer/submit-a-transaction
https://docs.polymarket.com/trading/bridge/deposit
https://docs.polymarket.com/market-data/websocket/overview
`
	report := CheckLLMS(text)
	if !report.OK {
		t.Fatalf("report should pass: %+v", report)
	}
	if len(report.Checked) == 0 {
		t.Fatal("expected checked surfaces")
	}
}

func TestCheckLLMSReportsMissingSurface(t *testing.T) {
	report := CheckLLMS("https://docs.polymarket.com/trading/overview")
	if report.OK {
		t.Fatalf("report should fail: %+v", report)
	}
	if !containsMissing(report, "gamma.markets") {
		t.Fatalf("missing gamma surface not reported: %+v", report.Missing)
	}
}

func TestCheckLLMSIgnoresMarkdownSuffixAndCase(t *testing.T) {
	text := `
HTTPS://DOCS.POLYMARKET.COM/MARKET-DATA/OVERVIEW.MD
https://docs.polymarket.com/TRADING/OVERVIEW.md
https://docs.polymarket.com/API-REFERENCE/CORE/GET-CURRENT-POSITIONS-FOR-A-USER.md
https://docs.polymarket.com/API-REFERENCE/RELAYER/SUBMIT-A-TRANSACTION.md
https://docs.polymarket.com/TRADING/BRIDGE/DEPOSIT.md
https://docs.polymarket.com/MARKET-DATA/WEBSOCKET/OVERVIEW.md
`
	if report := CheckLLMS(text); !report.OK {
		t.Fatalf("report should pass with .md/case variants: %+v", report)
	}
}

func containsMissing(report Report, id string) bool {
	for _, surface := range report.Missing {
		if surface.ID == id {
			return true
		}
	}
	return false
}
