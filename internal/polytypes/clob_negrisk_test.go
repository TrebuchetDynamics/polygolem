package polytypes

import (
	"encoding/json"
	"testing"
)

func TestNegRiskInfoDecodesStringFieldsAndCamelAliases(t *testing.T) {
	var row NegRiskInfo
	raw := `{"negRisk":"true","negRiskMarketID":"neg-market-1","negRiskFeeBips":"25"}`
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	if !row.NegRisk || row.NegRiskMarketID != "neg-market-1" || row.NegRiskFeeBips != 25 {
		t.Fatalf("row=%+v", row)
	}
}

func TestCLOBMarketDecodesCamelFeeDetailsWithStringFields(t *testing.T) {
	var row CLOBMarket
	raw := `{"condition_id":"condition-1","tokens":[],"feeDetails":{"rate":"0.01","exponent":"2","takerOnly":"true"}}`
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	if row.FeeDetails.Rate != 0.01 || row.FeeDetails.Exponent != 2 || !row.FeeDetails.TakerOnly {
		t.Fatalf("feeDetails=%+v", row.FeeDetails)
	}
}

func TestFeeRateDecodesStringFieldAndCamelAlias(t *testing.T) {
	var row FeeRate
	raw := `{"feeRateBps":"30"}`
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	if row.FeeRateBps != 30 {
		t.Fatalf("row=%+v", row)
	}
}

// TestFeeRateDecodesFloatFormattedBps guards the regression where a float-formatted
// bps string ("30.0") was silently parsed as 0 by strconv.Atoi.
func TestFeeRateDecodesFloatFormattedBps(t *testing.T) {
	cases := map[string]int{
		`{"fee_rate_bps":"30.0"}`: 30,
		`{"fee_rate_bps":"30.4"}`: 30,
		`{"fee_rate_bps":"30.6"}`: 31,
		`{"fee_rate_bps":"30"}`:   30,
		`{"fee_rate_bps":""}`:     0,
	}
	for raw, want := range cases {
		var row FeeRate
		if err := json.Unmarshal([]byte(raw), &row); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if row.FeeRateBps != want {
			t.Fatalf("%s: FeeRateBps=%d want %d", raw, row.FeeRateBps, want)
		}
	}
}
