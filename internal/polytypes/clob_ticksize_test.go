package polytypes

import (
	"encoding/json"
	"testing"
)

func TestTickSizeDecodesNumericFieldsAndCamelAliases(t *testing.T) {
	var row TickSize
	raw := `{"minimumTickSize":0.001,"minimumOrderSize":5,"tickSize":0.01}`
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	if row.MinimumTickSize != "0.001" || row.MinimumOrderSize != "5" || row.TickSize != "0.01" {
		t.Fatalf("row=%+v", row)
	}
}
