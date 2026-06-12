package tests

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/pkg/ctf"
	"github.com/ethereum/go-ethereum/common"
)

type clobAuthV2ConformanceFixture struct {
	SchemaVersion int                `json:"schema_version"`
	Family        string             `json:"family"`
	L1Vectors     []clobAuthVector   `json:"l1_vectors"`
	L2Vectors     []hmacHeaderVector `json:"l2_vectors"`
}

type builderHeadersConformanceFixture struct {
	SchemaVersion int                `json:"schema_version"`
	Family        string             `json:"family"`
	Vectors       []hmacHeaderVector `json:"vectors"`
}

type ctfCalldataConformanceFixture struct {
	SchemaVersion int    `json:"schema_version"`
	Family        string `json:"family"`
	Input         struct {
		CollateralToken    string   `json:"collateral_token"`
		ParentCollectionID string   `json:"parent_collection_id"`
		ConditionID        string   `json:"condition_id"`
		Partition          []string `json:"partition"`
		Amount             string   `json:"amount"`
	} `json:"input"`
	Vectors []struct {
		Name             string `json:"name"`
		Operation        string `json:"operation"`
		ExpectedSelector string `json:"expected_selector"`
		ExpectedCalldata string `json:"expected_calldata"`
	} `json:"vectors"`
}

func TestConformanceClobAuthV2FixtureVectors(t *testing.T) {
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "fixtures", "conformance", "clob_auth_v2.json"))
	if err != nil {
		t.Fatalf("read conformance ClobAuth fixture: %v", err)
	}
	var fixture clobAuthV2ConformanceFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode conformance ClobAuth fixture: %v", err)
	}
	if fixture.SchemaVersion != 1 || fixture.Family != "clob-auth-v2" {
		t.Fatalf("unexpected fixture identity: %+v", fixture)
	}
	if len(fixture.L1Vectors) == 0 || len(fixture.L2Vectors) == 0 {
		t.Fatalf("expected L1 and L2 vectors: %+v", fixture)
	}
	for _, vector := range fixture.L1Vectors {
		vector := vector
		t.Run("l1/"+vector.Name, func(t *testing.T) {
			got, err := auth.BuildL1HeadersFromPrivateKey(vector.PrivateKey, vector.ChainID, vector.Timestamp, vector.Nonce)
			if err != nil {
				t.Fatalf("build L1 headers: %v", err)
			}
			assertHeaders(t, got, vector.ExpectedHeaders)
		})
	}
	for _, vector := range fixture.L2Vectors {
		vector := vector
		t.Run("l2/"+vector.Name, func(t *testing.T) {
			got, err := auth.BuildL2Headers(&auth.APIKey{Key: vector.APIKey, Secret: vector.Secret, Passphrase: vector.Passphrase}, vector.Timestamp, vector.Method, vector.Path, vector.Body)
			if err != nil {
				t.Fatalf("build L2 headers: %v", err)
			}
			assertHeaders(t, got, vector.ExpectedHeaders)
		})
	}
}

func TestConformanceBuilderHeadersFixtureVectors(t *testing.T) {
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "fixtures", "conformance", "builder_headers.json"))
	if err != nil {
		t.Fatalf("read builder fixture: %v", err)
	}
	var fixture builderHeadersConformanceFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode builder fixture: %v", err)
	}
	if fixture.SchemaVersion != 1 || fixture.Family != "builder-attribution-headers" || len(fixture.Vectors) == 0 {
		t.Fatalf("unexpected builder fixture: %+v", fixture)
	}
	for _, vector := range fixture.Vectors {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			got, err := auth.BuildBuilderHeaders(&auth.BuilderConfig{Key: vector.APIKey, Secret: vector.Secret, Passphrase: vector.Passphrase}, vector.Timestamp, vector.Method, vector.Path, vector.Body)
			if err != nil {
				t.Fatalf("build builder headers: %v", err)
			}
			assertHeaders(t, got, vector.ExpectedHeaders)
		})
	}
}

func TestConformanceCTFFullCalldataVectors(t *testing.T) {
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "fixtures", "conformance", "ctf_calldata.json"))
	if err != nil {
		t.Fatalf("read CTF calldata fixture: %v", err)
	}
	var fixture ctfCalldataConformanceFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode CTF calldata fixture: %v", err)
	}
	if fixture.SchemaVersion != 1 || fixture.Family != "ctf-calldata" || len(fixture.Vectors) == 0 {
		t.Fatalf("unexpected CTF fixture: %+v", fixture)
	}
	partition := make([]*big.Int, len(fixture.Input.Partition))
	for i, raw := range fixture.Input.Partition {
		value, ok := new(big.Int).SetString(raw, 10)
		if !ok {
			t.Fatalf("bad partition[%d]=%q", i, raw)
		}
		partition[i] = value
	}
	amount, ok := new(big.Int).SetString(fixture.Input.Amount, 10)
	if !ok {
		t.Fatalf("bad amount %q", fixture.Input.Amount)
	}
	collateral := common.HexToAddress(fixture.Input.CollateralToken)
	parent := common.HexToHash(fixture.Input.ParentCollectionID)
	condition := common.HexToHash(fixture.Input.ConditionID)
	for _, vector := range fixture.Vectors {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			var data []byte
			var err error
			switch vector.Operation {
			case "split":
				data, err = ctf.SplitPositionData(collateral, parent, condition, partition, amount)
			case "merge":
				data, err = ctf.MergePositionsData(collateral, parent, condition, partition, amount)
			case "redeem":
				data, err = ctf.RedeemPositionsData(collateral, parent, condition, partition)
			default:
				t.Fatalf("unknown operation %q", vector.Operation)
			}
			if err != nil {
				t.Fatalf("build calldata: %v", err)
			}
			got := "0x" + hex.EncodeToString(data)
			if got[:10] != vector.ExpectedSelector {
				t.Fatalf("selector mismatch got %s want %s", got[:10], vector.ExpectedSelector)
			}
			if got != vector.ExpectedCalldata {
				t.Fatalf("calldata mismatch\ngot:  %s\nwant: %s", got, vector.ExpectedCalldata)
			}
		})
	}
}

func assertHeaders(t *testing.T, got, want map[string]string) {
	t.Helper()
	for key, expected := range want {
		if got[key] != expected {
			t.Fatalf("%s mismatch\ngot:  %q\nwant: %q", key, got[key], expected)
		}
	}
}
