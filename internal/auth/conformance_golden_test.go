package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type goldenClobAuthV2Fixture struct {
	SchemaVersion int                `json:"schema_version"`
	Family        string             `json:"family"`
	L1Vectors     []goldenL1Vector   `json:"l1_vectors"`
	L2Vectors     []goldenHMACVector `json:"l2_vectors"`
}

type goldenBuilderHeadersFixture struct {
	SchemaVersion int                `json:"schema_version"`
	Family        string             `json:"family"`
	Vectors       []goldenHMACVector `json:"vectors"`
}

type goldenL1Vector struct {
	Name            string            `json:"name"`
	PrivateKey      string            `json:"private_key"`
	ChainID         int64             `json:"chain_id"`
	Timestamp       int64             `json:"timestamp"`
	Nonce           int64             `json:"nonce"`
	ExpectedHeaders map[string]string `json:"expected_headers"`
}

type goldenHMACVector struct {
	Name            string            `json:"name"`
	APIKey          string            `json:"api_key"`
	Secret          string            `json:"secret"`
	Passphrase      string            `json:"passphrase"`
	Timestamp       int64             `json:"timestamp"`
	Method          string            `json:"method"`
	Path            string            `json:"path"`
	Body            *string           `json:"body"`
	ExpectedHeaders map[string]string `json:"expected_headers"`
}

func TestGoldenConformanceClobAuthV2Headers(t *testing.T) {
	raw := readGoldenFixture(t, "clob_auth_v2.json")
	var fixture goldenClobAuthV2Fixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if fixture.SchemaVersion != 1 || fixture.Family != "clob-auth-v2" {
		t.Fatalf("unexpected fixture identity: %+v", fixture)
	}
	for _, vector := range fixture.L1Vectors {
		vector := vector
		t.Run("l1/"+vector.Name, func(t *testing.T) {
			got, err := BuildL1HeadersFromPrivateKey(vector.PrivateKey, vector.ChainID, vector.Timestamp, vector.Nonce)
			if err != nil {
				t.Fatalf("build L1 headers: %v", err)
			}
			assertGoldenHeaders(t, got, vector.ExpectedHeaders)
		})
	}
	for _, vector := range fixture.L2Vectors {
		vector := vector
		t.Run("l2/"+vector.Name, func(t *testing.T) {
			got, err := BuildL2Headers(&APIKey{Key: vector.APIKey, Secret: vector.Secret, Passphrase: vector.Passphrase}, vector.Timestamp, vector.Method, vector.Path, vector.Body)
			if err != nil {
				t.Fatalf("build L2 headers: %v", err)
			}
			assertGoldenHeaders(t, got, vector.ExpectedHeaders)
		})
	}
}

func TestGoldenConformanceBuilderHeaders(t *testing.T) {
	raw := readGoldenFixture(t, "builder_headers.json")
	var fixture goldenBuilderHeadersFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if fixture.SchemaVersion != 1 || fixture.Family != "builder-attribution-headers" {
		t.Fatalf("unexpected fixture identity: %+v", fixture)
	}
	for _, vector := range fixture.Vectors {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			got, err := BuildBuilderHeaders(&BuilderConfig{Key: vector.APIKey, Secret: vector.Secret, Passphrase: vector.Passphrase}, vector.Timestamp, vector.Method, vector.Path, vector.Body)
			if err != nil {
				t.Fatalf("build builder headers: %v", err)
			}
			assertGoldenHeaders(t, got, vector.ExpectedHeaders)
		})
	}
}

func readGoldenFixture(t *testing.T, name string) []byte {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(wd, "..", "..", "fixtures", "conformance", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return raw
}

func assertGoldenHeaders(t *testing.T, got, want map[string]string) {
	t.Helper()
	for key, expected := range want {
		if got[key] != expected {
			t.Fatalf("%s mismatch\ngot:  %q\nwant: %q", key, got[key], expected)
		}
	}
}
