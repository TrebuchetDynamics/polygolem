package types

import (
	"encoding/json"
	"testing"
)

func TestCLOBTickSizeDecodesNumericFieldsAndCamelAliases(t *testing.T) {
	var row CLOBTickSize
	raw := `{"minimumTickSize":0.001,"minimumOrderSize":5,"tickSize":0.01}`
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	if row.MinimumTickSize != "0.001" || row.MinimumOrderSize != "5" || row.TickSize != "0.01" {
		t.Fatalf("row=%+v", row)
	}
}
