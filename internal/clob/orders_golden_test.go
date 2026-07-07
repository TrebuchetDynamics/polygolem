package clob

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// goldenFixture pins one neg-risk variant of the sigtype-3 (POLY_1271,
// deposit wallet) signing path. Since the 2026-04-28 V2 cutover, sigtypes
// 0/1/2 are dead — `clob/order` rejects them — so we only pin sigtype 3.
//
// expectedHash and expectedSig are populated by running the test once with
// placeholder values, capturing the actual outputs, and pasting them in.
// They serve as regression pins: if the V2 wire format drifts (struct
// fields, EIP-712 domain, signing logic), these tests fail and force the
// change to be deliberate.
type goldenFixture struct {
	name         string
	negRisk      bool
	expectedHash string // 0x-prefixed 64-hex EIP-712 struct hash
	expectedSig  string // 0x-prefixed POLY_1271 wrapped signature (636 chars)
}

var goldenFixtures = []goldenFixture{
	{
		name:         "poly1271_regular",
		negRisk:      false,
		expectedHash: "0xf45fbc10003ca8571f7a6653a450938f99a36b21940ec401c9da0a4065d35fb6",
		expectedSig:  "0x4625c88733f6fa889061fd94fd47bb25d142a429026e3be78dc91caaaa59886d6019105a18824f989f7f77a2396864aa4f3dee83e78347828e2fe92cb4b3548b1b3264e159346253e26a64e00b69032db0e7d32f94628de3e6eecb50304d7af3d2f45fbc10003ca8571f7a6653a450938f99a36b21940ec401c9da0a4065d35fb64f726465722875696e743235362073616c742c61646472657373206d616b65722c61646472657373207369676e65722c75696e7432353620746f6b656e49642c75696e74323536206d616b6572416d6f756e742c75696e743235362074616b6572416d6f756e742c75696e743820736964652c75696e7438207369676e6174757265547970652c75696e743235362074696d657374616d702c62797465733332206d657461646174612c62797465733332206275696c6465722900ba",
	},
	{
		name:    "poly1271_neg_risk",
		negRisk: true,
		// expectedHash and expectedSig must be populated by running the test
		// once, capturing output, and pasting the verified values here.
		expectedHash: "0xf45fbc10003ca8571f7a6653a450938f99a36b21940ec401c9da0a4065d35fb6",
		expectedSig:  "0x4566e121b1e96452bedd7991ee4683ecebdd3873b240f794b78643cffc8079a80d44773dc2b8baee18e7ea6189d0ee43d1287e67a93472ce8e052497e1a4da6c1b9b858f53327b0bd13af8ec14cfb35234fb9eb7b0504d1a4e61f433840d30e81af45fbc10003ca8571f7a6653a450938f99a36b21940ec401c9da0a4065d35fb64f726465722875696e743235362073616c742c61646472657373206d616b65722c61646472657373207369676e65722c75696e7432353620746f6b656e49642c75696e74323536206d616b6572416d6f756e742c75696e743235362074616b6572416d6f756e742c75696e743820736964652c75696e7438207369676e6174757265547970652c75696e743235362074696d657374616d702c62797465733332206d657461646174612c62797465733332206275696c6465722900ba",
	},
}

func TestGoldenVectorsV2OrderSigning(t *testing.T) {
	origSalt := orderSalt
	origNow := orderNow
	t.Cleanup(func() {
		orderSalt = origSalt
		orderNow = origNow
	})
	orderSalt = func() (uint64, error) { return 1, nil }
	orderNow = func() time.Time { return time.UnixMilli(1778125000123) }

	signer, err := auth.NewPrivateKeySigner(testOrderPrivateKey, polygonChainID)
	if err != nil {
		t.Fatal(err)
	}

	for _, fx := range goldenFixtures {
		t.Run(fx.name, func(t *testing.T) {
			payload, err := buildSignedOrderPayload(signer, orderDraft{
				tokenID:     big.NewInt(12345),
				side:        "BUY",
				makerAmount: "700000",
				takerAmount: "1400000",
				orderType:   "GTC",
			}, time.UnixMilli(1778125000123), fx.negRisk)
			if err != nil {
				t.Fatal(err)
			}

			// Compute the EIP-712 typed-data hash for the order.
			typedData, err := buildOrderTypedData(payload, fx.negRisk)
			if err != nil {
				t.Fatalf("build typed data: %v", err)
			}
			_, rawDataStr, err := apitypes.TypedDataAndHash(typedData)
			if err != nil {
				t.Fatalf("typed-data hash: %v", err)
			}
			rawData := []byte(rawDataStr)
			gotHash := "0x" + hex.EncodeToString(rawData[34:66]) // structHash slice

			if gotHash != fx.expectedHash {
				t.Fatalf("hash mismatch for %s:\n  got  %s\n  want %s", fx.name, gotHash, fx.expectedHash)
			}
			if payload.Signature != fx.expectedSig {
				t.Fatalf("signature mismatch for %s:\n  got  %s\n  want %s", fx.name, payload.Signature, fx.expectedSig)
			}
		})
	}
}
