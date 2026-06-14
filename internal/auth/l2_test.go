package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"testing"
)

func TestSignHMACDecodesURLSafeBase64Secrets(t *testing.T) {
	secretBytes := []byte{251, 255, 255, 1, 2, 3}
	secret := base64.URLEncoding.EncodeToString(secretBytes)
	timestamp := int64(1000000)
	method := "GET"
	path := "/balance-allowance"

	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(strconv.FormatInt(timestamp, 10) + method + path))
	want := base64.URLEncoding.EncodeToString(mac.Sum(nil))

	if got := SignHMAC(secret, timestamp, method, path, nil); got != want {
		t.Fatalf("signature=%q, want %q", got, want)
	}
}

// TestCompactJSONPreservesSpacesInsideStrings guards the regression where the
// hand-rolled scanner desynced on escaped quotes and stripped legitimate spaces
// inside JSON string values, making the HMAC-signed body differ from the bytes
// actually sent.
func TestCompactJSONPreservesSpacesInsideStrings(t *testing.T) {
	cases := map[string]string{
		`{"a": 1, "b": 2}`:            `{"a":1,"b":2}`,
		`{"msg":"a \"b c\" d"}`:       `{"msg":"a \"b c\" d"}`,
		`{ "orderID" : "0xAbC 123" }`: `{"orderID":"0xAbC 123"}`,
		`{"orderIDs":["a b","c d"]}`:  `{"orderIDs":["a b","c d"]}`,
	}
	for in, want := range cases {
		if got := CompactJSON(in); got != want {
			t.Fatalf("CompactJSON(%q)=%q want %q", in, got, want)
		}
	}
	// Invalid JSON is returned unchanged rather than corrupted.
	if got := CompactJSON("not json"); got != "not json" {
		t.Fatalf("CompactJSON(invalid)=%q want unchanged", got)
	}
}
