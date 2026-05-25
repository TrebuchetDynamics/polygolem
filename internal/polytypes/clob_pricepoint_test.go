package polytypes

import (
	"encoding/json"
	"testing"
)

func TestPricePointDecodesLongAliasesAndNumericFields(t *testing.T) {
	var point PricePoint
	raw := `{"timestamp":1714000000,"price":0.51,"volume":12,"interval":"1m"}`
	if err := json.Unmarshal([]byte(raw), &point); err != nil {
		t.Fatal(err)
	}
	if point.T != "1714000000" || point.P != "0.51" || point.Volume != "12" || point.Interval != "1m" {
		t.Fatalf("point=%+v", point)
	}
}
