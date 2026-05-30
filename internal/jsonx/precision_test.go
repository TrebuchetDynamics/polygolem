package jsonx

import (
	"encoding/json"
	"testing"
)

func TestStringOrNumberLargeIntegerPrecision(t *testing.T) {
	// uint256 token IDs can exceed float64's 53-bit mantissa
	tests := map[string]string{
		`10000000000000001`:      "10000000000000001",
		`999999999999999999`:     "999999999999999999",
		`9223372036854775807`:    "9223372036854775807",
		`12345678901234567890`:   "12345678901234567890",
		`"10000000000000001"`:    "10000000000000001",
		`"0.00000001"`:          "0.00000001",
	}
	for raw, want := range tests {
		if got := StringOrNumber(json.RawMessage(raw)); got != want {
			t.Errorf("StringOrNumber(%s) = %q, want %q", raw, got, want)
		}
	}
}
