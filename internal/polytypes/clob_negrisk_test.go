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
