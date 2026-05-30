// Package jsonx contains narrow JSON decoding helpers for Polymarket API
// responses that drift between strings, numbers, booleans, nulls, and empty
// fields.
package jsonx

import (
	"encoding/json"
	"strconv"
	"strings"
)

// StringOrNumber decodes a JSON scalar into its stable string representation.
func StringOrNumber(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(string(raw))
	}
}

// FirstString returns the first non-empty string value.
func FirstString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// FirstNonBlankString returns the first string whose trimmed value is non-empty.
// It preserves the original string value so callers keep existing formatting.
func FirstNonBlankString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// FirstTrimmedString returns the first non-blank string after trimming it.
func FirstTrimmedString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// FirstStringOrNumber returns the first non-empty StringOrNumber value.
func FirstStringOrNumber(values ...json.RawMessage) string {
	for _, value := range values {
		if decoded := StringOrNumber(value); decoded != "" {
			return decoded
		}
	}
	return ""
}

// FirstRaw returns the first present, non-null raw JSON value.
func FirstRaw(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		text := strings.TrimSpace(string(value))
		if len(value) == 0 || text == "" || text == "null" {
			continue
		}
		return value
	}
	return nil
}

// BoolOrFalse decodes bool-like API fields; missing, null, and unknown values are false.
func BoolOrFalse(raw json.RawMessage) bool {
	switch strings.ToLower(StringOrNumber(raw)) {
	case "true", "1":
		return true
	default:
		return false
	}
}
