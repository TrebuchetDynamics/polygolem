package clob

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type orderProtocolFixture struct {
	SchemaVersion int    `json:"schema_version"`
	Family        string `json:"family"`
	Source        struct {
		ReferencePaths []string `json:"reference_paths"`
		Notes          string   `json:"notes"`
	} `json:"source"`
	Input struct {
		PrivateKey      string `json:"private_key"`
		Salt            uint64 `json:"salt"`
		TimestampMillis int64  `json:"timestamp_millis"`
		TokenID         string `json:"token_id"`
		Side            string `json:"side"`
		MakerAmount     string `json:"maker_amount"`
		TakerAmount     string `json:"taker_amount"`
		OrderType       string `json:"order_type"`
	} `json:"input"`
	Vectors []struct {
		Name               string `json:"name"`
		NegRisk            bool   `json:"neg_risk"`
		ExpectedStructHash string `json:"expected_struct_hash"`
		ExpectedSignature  string `json:"expected_signature"`
	} `json:"vectors"`
}

func TestProtocolFixtureV2OrderSigning(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(wd, "..", "..", "fixtures", "protocol", "eip712_orders.json"))
	if err != nil {
		t.Fatalf("read order fixture: %v", err)
	}
	var fixture orderProtocolFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode order fixture: %v", err)
	}
	if fixture.SchemaVersion != 1 || fixture.Family != "v2-poly1271-order-eip712" {
		t.Fatalf("unexpected fixture identity: %+v", fixture)
	}
	if len(fixture.Source.ReferencePaths) == 0 || fixture.Source.Notes == "" {
		t.Fatalf("fixture must record source provenance: %+v", fixture.Source)
	}

	origSalt := orderSalt
	origNow := orderNow
	t.Cleanup(func() {
		orderSalt = origSalt
		orderNow = origNow
	})
	orderSalt = func() (uint64, error) { return fixture.Input.Salt, nil }
	orderNow = func() time.Time { return time.UnixMilli(fixture.Input.TimestampMillis) }

	signer, err := auth.NewPrivateKeySigner(fixture.Input.PrivateKey, polygonChainID)
	if err != nil {
		t.Fatal(err)
	}
	tokenID, ok := new(big.Int).SetString(fixture.Input.TokenID, 10)
	if !ok {
		t.Fatalf("bad fixture token_id %q", fixture.Input.TokenID)
	}

	for _, vector := range fixture.Vectors {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			payload, err := buildSignedOrderPayload(signer, orderDraft{
				tokenID:     tokenID,
				side:        fixture.Input.Side,
				makerAmount: fixture.Input.MakerAmount,
				takerAmount: fixture.Input.TakerAmount,
				orderType:   fixture.Input.OrderType,
			}, time.UnixMilli(fixture.Input.TimestampMillis), vector.NegRisk)
			if err != nil {
				t.Fatal(err)
			}
			typedData, err := buildOrderTypedData(payload, vector.NegRisk)
			if err != nil {
				t.Fatalf("build typed data: %v", err)
			}
			_, rawDataStr, err := apitypes.TypedDataAndHash(typedData)
			if err != nil {
				t.Fatalf("typed-data hash: %v", err)
			}
			rawData := []byte(rawDataStr)
			gotHash := "0x" + hex.EncodeToString(rawData[34:66])
			if gotHash != vector.ExpectedStructHash {
				t.Fatalf("hash mismatch\ngot:  %s\nwant: %s", gotHash, vector.ExpectedStructHash)
			}
			if payload.Signature != vector.ExpectedSignature {
				t.Fatalf("signature mismatch\ngot:  %s\nwant: %s", payload.Signature, vector.ExpectedSignature)
			}
		})
	}
}
