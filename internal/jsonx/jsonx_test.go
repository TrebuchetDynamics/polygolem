package jsonx

import (
	"encoding/json"
	"testing"
)

func TestStringOrNumberDecodesScalarDrift(t *testing.T) {
	tests := map[string]string{
		`" token "`: "token",
		`12345`:     "12345",
		`0.125`:     "0.125",
		`true`:      "true",
		`false`:     "false",
		`null`:      "",
		``:          "",
	}
	for raw, want := range tests {
		if got := StringOrNumber(json.RawMessage(raw)); got != want {
			t.Fatalf("StringOrNumber(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestFirstStringSkipsEmptyValues(t *testing.T) {
	got := FirstString("", "winner")
	if got != "winner" {
		t.Fatalf("FirstString = %q, want winner", got)
	}
}

func TestFirstNonBlankStringSkipsWhitespaceValues(t *testing.T) {
	got := FirstNonBlankString("", "  ", " kept ")
	if got != " kept " {
		t.Fatalf("FirstNonBlankString = %q, want original non-blank value", got)
	}
}

func TestFirstTrimmedStringSkipsWhitespaceAndTrimsValue(t *testing.T) {
	got := FirstTrimmedString("", "  ", " kept ")
	if got != "kept" {
		t.Fatalf("FirstTrimmedString = %q, want kept", got)
	}
}

func TestFirstStringOrNumberSkipsEmptyValues(t *testing.T) {
	got := FirstStringOrNumber(json.RawMessage(`null`), json.RawMessage(`""`), json.RawMessage(`42`))
	if got != "42" {
		t.Fatalf("FirstStringOrNumber = %q, want 42", got)
	}
}

func TestFirstRawSkipsMissingAndNullValues(t *testing.T) {
	got := FirstRaw(nil, json.RawMessage(`null`), json.RawMessage(`"kept"`))
	if string(got) != `"kept"` {
		t.Fatalf("FirstRaw = %s, want %s", got, `"kept"`)
	}
}

func TestBoolOrFalseDecodesBoolDrift(t *testing.T) {
	for _, raw := range []string{`true`, `"true"`, `1`, `"1"`} {
		if !BoolOrFalse(json.RawMessage(raw)) {
			t.Fatalf("BoolOrFalse(%s) = false, want true", raw)
		}
	}
	for _, raw := range []string{`false`, `"false"`, `0`, `"0"`, `null`, `""`} {
		if BoolOrFalse(json.RawMessage(raw)) {
			t.Fatalf("BoolOrFalse(%s) = true, want false", raw)
		}
	}
}
