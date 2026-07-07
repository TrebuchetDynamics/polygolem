// Package upstreamdrift checks saved official Polymarket docs indexes against
// the Polygolem compatibility surface.
package upstreamdrift

import "strings"

type Surface struct {
	ID       string `json:"id"`
	Service  string `json:"service"`
	Expected string `json:"expected"`
}

type Report struct {
	OK      bool      `json:"ok"`
	Checked []Surface `json:"checked"`
	Missing []Surface `json:"missing"`
}

func CheckLLMS(text string) Report {
	normalized := normalize(text)
	report := Report{OK: true, Checked: expectedSurfaces()}
	for _, surface := range report.Checked {
		if !strings.Contains(normalized, normalize(surface.Expected)) {
			report.Missing = append(report.Missing, surface)
		}
	}
	report.OK = len(report.Missing) == 0
	return report
}

func expectedSurfaces() []Surface {
	return []Surface{
		{ID: "gamma.markets", Service: "Gamma API", Expected: "https://docs.polymarket.com/market-data/overview"},
		{ID: "clob.public_data", Service: "CLOB API", Expected: "https://docs.polymarket.com/trading/overview"},
		{ID: "data.positions", Service: "Data API", Expected: "https://docs.polymarket.com/api-reference/core/get-current-positions-for-a-user"},
		{ID: "relayer.deposit_wallet", Service: "Relayer V2", Expected: "https://docs.polymarket.com/api-reference/relayer/submit-a-transaction"},
		{ID: "bridge.funding", Service: "Bridge", Expected: "https://docs.polymarket.com/trading/bridge/deposit"},
		{ID: "websocket.market", Service: "CLOB WebSocket", Expected: "https://docs.polymarket.com/market-data/websocket/overview"},
	}
}

func normalize(value string) string {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, ".md", "")
	return strings.TrimSpace(value)
}
