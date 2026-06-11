package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/ethereum/go-ethereum/crypto"
)

type clobAuthFixture struct {
	SchemaVersion int    `json:"schema_version"`
	Family        string `json:"family"`
	Source        struct {
		ReferencePaths []string `json:"reference_paths"`
		Notes          string   `json:"notes"`
	} `json:"source"`
	Vectors []clobAuthVector `json:"vectors"`
}

type clobAuthVector struct {
	Name            string            `json:"name"`
	PrivateKey      string            `json:"private_key"`
	ChainID         int64             `json:"chain_id"`
	Timestamp       int64             `json:"timestamp"`
	Nonce           int64             `json:"nonce"`
	ExpectedHeaders map[string]string `json:"expected_headers"`
}

type hmacHeaderFixture struct {
	SchemaVersion int    `json:"schema_version"`
	Family        string `json:"family"`
	Source        struct {
		Formula        string   `json:"formula"`
		ReferencePaths []string `json:"reference_paths"`
		Notes          string   `json:"notes"`
	} `json:"source"`
	Vectors []hmacHeaderVector `json:"vectors"`
}

type hmacHeaderVector struct {
	Name            string            `json:"name"`
	Kind            string            `json:"kind"`
	APIKey          string            `json:"api_key"`
	Secret          string            `json:"secret"`
	Passphrase      string            `json:"passphrase"`
	Timestamp       int64             `json:"timestamp"`
	Method          string            `json:"method"`
	Path            string            `json:"path"`
	Body            *string           `json:"body"`
	ExpectedHeaders map[string]string `json:"expected_headers"`
}

type ctfCalldataFixture struct {
	SchemaVersion int    `json:"schema_version"`
	Family        string `json:"family"`
	Source        struct {
		ReferencePaths []string `json:"reference_paths"`
		Notes          string   `json:"notes"`
	} `json:"source"`
	Vectors []ctfSelectorVector `json:"vectors"`
}

type ctfSelectorVector struct {
	Name             string `json:"name"`
	Signature        string `json:"signature"`
	ExpectedSelector string `json:"expected_selector"`
}

type depositWalletBatchFixture struct {
	SchemaVersion int               `json:"schema_version"`
	Family        string            `json:"family"`
	Expected      map[string]string `json:"expected"`
	Source        struct {
		ReferencePaths []string `json:"reference_paths"`
		Notes          string   `json:"notes"`
	} `json:"source"`
}

func TestProtocolConformanceClobAuthL1Headers(t *testing.T) {
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "fixtures", "protocol", "clob_auth.json"))
	if err != nil {
		t.Fatalf("read ClobAuth fixture: %v", err)
	}
	var fixture clobAuthFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode ClobAuth fixture: %v", err)
	}
	if fixture.SchemaVersion != 1 || fixture.Family != "clob-auth-l1-headers" {
		t.Fatalf("unexpected ClobAuth fixture identity: %+v", fixture)
	}
	if len(fixture.Source.ReferencePaths) == 0 || fixture.Source.Notes == "" {
		t.Fatalf("fixture must record source provenance: %+v", fixture.Source)
	}
	for _, vector := range fixture.Vectors {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			got, err := auth.BuildL1HeadersFromPrivateKey(vector.PrivateKey, vector.ChainID, vector.Timestamp, vector.Nonce)
			if err != nil {
				t.Fatalf("build L1 headers: %v", err)
			}
			for key, want := range vector.ExpectedHeaders {
				if got[key] != want {
					t.Fatalf("%s mismatch\ngot:  %q\nwant: %q", key, got[key], want)
				}
			}
			if len(got["POLY_SIGNATURE"]) != 132 {
				t.Fatalf("POLY_SIGNATURE length=%d want 132 for 65-byte EOA signature", len(got["POLY_SIGNATURE"]))
			}
		})
	}
}

func TestProtocolConformanceDepositWalletBatchTypedData(t *testing.T) {
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "fixtures", "protocol", "deposit_wallet_batch.json"))
	if err != nil {
		t.Fatalf("read deposit-wallet batch fixture: %v", err)
	}
	var fixture depositWalletBatchFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode deposit-wallet batch fixture: %v", err)
	}
	if fixture.SchemaVersion != 1 || fixture.Family != "deposit-wallet-batch-typed-data" {
		t.Fatalf("unexpected fixture identity: %+v", fixture)
	}
	if len(fixture.Source.ReferencePaths) == 0 || fixture.Source.Notes == "" {
		t.Fatalf("fixture must record source provenance: %+v", fixture.Source)
	}

	cmd := exec.Command("go", "run", "./cmd/parity_walletbatch")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run parity_walletbatch: %v\n%s", err, out)
	}
	var got map[string]string
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decode parity output: %v\n%s", err, out)
	}
	for key, want := range fixture.Expected {
		if got[key] != want {
			t.Fatalf("%s mismatch\ngot:  %s\nwant: %s", key, got[key], want)
		}
	}
}

func TestProtocolConformanceCTFCalldataSelectors(t *testing.T) {
	root := repositoryRoot(t)
	fixturePath := filepath.Join(root, "fixtures", "protocol", "ctf_calldata.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read CTF fixture: %v", err)
	}
	var fixture ctfCalldataFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode CTF fixture: %v", err)
	}
	if fixture.SchemaVersion != 1 || fixture.Family != "ctf-calldata-selectors" {
		t.Fatalf("unexpected CTF fixture identity: %+v", fixture)
	}
	if len(fixture.Source.ReferencePaths) == 0 || fixture.Source.Notes == "" {
		t.Fatalf("fixture must record reference paths and notes: %+v", fixture.Source)
	}
	if len(fixture.Vectors) == 0 {
		t.Fatal("expected CTF selector vectors")
	}
	for _, vector := range fixture.Vectors {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			selector := "0x" + crypto.Keccak256Hash([]byte(vector.Signature)).Hex()[2:10]
			if selector != vector.ExpectedSelector {
				t.Fatalf("selector mismatch for %s\ngot:  %s\nwant: %s", vector.Signature, selector, vector.ExpectedSelector)
			}
		})
	}
}

func TestProtocolConformanceHMACHeaders(t *testing.T) {
	root := repositoryRoot(t)
	fixturePath := filepath.Join(root, "fixtures", "protocol", "hmac_headers.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read HMAC fixture: %v", err)
	}
	var fixture hmacHeaderFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode HMAC fixture: %v", err)
	}
	if fixture.SchemaVersion != 1 {
		t.Fatalf("schema_version=%d want 1", fixture.SchemaVersion)
	}
	if fixture.Family != "polymarket-hmac-headers" {
		t.Fatalf("family=%q", fixture.Family)
	}
	if fixture.Source.Formula == "" || len(fixture.Source.ReferencePaths) == 0 || fixture.Source.Notes == "" {
		t.Fatalf("fixture must record formula, reference paths, and notes: %+v", fixture.Source)
	}
	if len(fixture.Vectors) == 0 {
		t.Fatal("expected at least one HMAC vector")
	}

	for _, vector := range fixture.Vectors {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			var (
				got map[string]string
				err error
			)
			switch vector.Kind {
			case "l2":
				got, err = auth.BuildL2Headers(&auth.APIKey{
					Key:        vector.APIKey,
					Secret:     vector.Secret,
					Passphrase: vector.Passphrase,
				}, vector.Timestamp, vector.Method, vector.Path, vector.Body)
			case "builder":
				got, err = auth.BuildBuilderHeaders(&auth.BuilderConfig{
					Key:        vector.APIKey,
					Secret:     vector.Secret,
					Passphrase: vector.Passphrase,
				}, vector.Timestamp, vector.Method, vector.Path, vector.Body)
			default:
				t.Fatalf("unsupported vector kind %q", vector.Kind)
			}
			if err != nil {
				t.Fatalf("build headers: %v", err)
			}
			for key, want := range vector.ExpectedHeaders {
				if got[key] != want {
					t.Fatalf("%s mismatch\ngot:  %q\nwant: %q", key, got[key], want)
				}
			}
		})
	}
}
