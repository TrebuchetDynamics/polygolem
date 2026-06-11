package httpsigner

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/signers"
)

func TestHTTPSignerSignsHashThroughRemoteService(t *testing.T) {
	var got map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"signature": "0x" + strings.Repeat("11", 65)})
	}))
	defer server.Close()

	signer, err := New(Config{URL: server.URL, BearerToken: "test-token", Address: "0xabc", ChainID: 137})
	if err != nil {
		t.Fatal(err)
	}
	var _ signers.Signer = signer

	var hash [32]byte
	copy(hash[:], []byte("fixture-hash-32-byte-value-0000"))
	sig, err := signer.SignHash(hash)
	if err != nil {
		t.Fatalf("sign hash: %v", err)
	}
	if len(sig) != 65 {
		t.Fatalf("signature length=%d", len(sig))
	}
	if got["operation"] != "sign_hash" || got["address"] != "0xabc" || int64(got["chain_id"].(float64)) != 137 {
		t.Fatalf("unexpected payload: %#v", got)
	}
	if gotHash, ok := got["hash"].(string); !ok || !strings.HasPrefix(gotHash, "0x") {
		t.Fatalf("missing hex hash in payload: %#v", got)
	}
}

func TestHTTPSignerSignsTypedDataHashes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if got["operation"] != "sign_typed_data_hashes" {
			t.Fatalf("operation=%v", got["operation"])
		}
		if got["domain_hash"] == "" || got["struct_hash"] == "" {
			t.Fatalf("missing hashes: %#v", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"result": "0x" + strings.Repeat("22", 32)})
	}))
	defer server.Close()

	signer, err := New(Config{URL: server.URL, BearerToken: "test-token"})
	if err != nil {
		t.Fatal(err)
	}
	var domainHash, structHash [32]byte
	got, err := signer.SignTypedData(domainHash, structHash)
	if err != nil {
		t.Fatalf("sign typed data: %v", err)
	}
	if got[0] != 0x22 || got[31] != 0x22 {
		t.Fatalf("unexpected typed-data result: %x", got)
	}
}

func TestHTTPSignerRejectsMissingConfig(t *testing.T) {
	if _, err := New(Config{URL: "", BearerToken: "token"}); err == nil {
		t.Fatal("expected URL validation error")
	}
	if _, err := New(Config{URL: "http://example.invalid", BearerToken: ""}); err == nil {
		t.Fatal("expected bearer token validation error")
	}
}

func TestHTTPSignerErrorsDoNotLeakBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "denied", http.StatusUnauthorized)
	}))
	defer server.Close()

	signer, err := New(Config{URL: server.URL, BearerToken: "super-secret-token", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	var hash [32]byte
	_, err = signer.SignHash(hash)
	if err == nil {
		t.Fatal("expected remote signer error")
	}
	if strings.Contains(err.Error(), "super-secret-token") {
		t.Fatalf("error leaked bearer token: %v", err)
	}
}
