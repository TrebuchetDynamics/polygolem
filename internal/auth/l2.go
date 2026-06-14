package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// BuildL2Headers builds the L2 HMAC authentication headers.
// Signature = base64url(HMAC-SHA256(decoded_secret, timestamp + method + path + body))
// Body must be compact JSON (no spaces).
func BuildL2Headers(apiKey *APIKey, timestamp int64, method, path string, body *string) (map[string]string, error) {
	if err := apiKey.Validate(); err != nil {
		return nil, err
	}
	sig := SignHMAC(apiKey.Secret, timestamp, method, path, body)
	return map[string]string{
		"POLY_API_KEY":    apiKey.Key,
		"POLY_PASSPHRASE": apiKey.Passphrase,
		"POLY_TIMESTAMP":  strconv.FormatInt(timestamp, 10),
		"POLY_SIGNATURE":  sig,
	}, nil
}

// SignHMAC computes a Polymarket-compatible HMAC signature.
func SignHMAC(secret string, timestamp int64, method, path string, body *string) string {
	key, err := decodeHMACSecret(secret)
	if err != nil {
		// Fallback: use raw secret bytes
		key = []byte(secret)
	}
	message := strconv.FormatInt(timestamp, 10) + method + path
	if body != nil {
		message += *body
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	// Convert to URL-safe base64
	sig = strings.ReplaceAll(sig, "+", "-")
	sig = strings.ReplaceAll(sig, "/", "_")
	return sig
}

func decodeHMACSecret(secret string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{
		base64.URLEncoding,
		base64.RawURLEncoding,
		base64.StdEncoding,
		base64.RawStdEncoding,
	} {
		key, err := enc.DecodeString(secret)
		if err == nil {
			return key, nil
		}
	}
	return nil, fmt.Errorf("invalid base64 secret")
}

// BuildBuilderHeaders builds builder attribution headers.
// Separate from L2 user credentials per PRD R3.
func BuildBuilderHeaders(bc *BuilderConfig, timestamp int64, method, path string, body *string) (map[string]string, error) {
	if !bc.Valid() {
		return nil, fmt.Errorf("builder config incomplete")
	}
	sig := SignHMAC(bc.Secret, timestamp, method, path, body)
	return map[string]string{
		"POLY_BUILDER_API_KEY":    bc.Key,
		"POLY_BUILDER_PASSPHRASE": bc.Passphrase,
		"POLY_BUILDER_TIMESTAMP":  strconv.FormatInt(timestamp, 10),
		"POLY_BUILDER_SIGNATURE":  sig,
	}, nil
}

// CompactJSON removes insignificant whitespace from a JSON string for HMAC
// body signing. It uses the escape-aware encoding/json.Compact so that spaces
// inside string values (including those preceded by escaped quotes) are
// preserved; the signed body therefore matches the bytes that are actually
// sent. If the input is not valid JSON it is returned unchanged.
func CompactJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}
