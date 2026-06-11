package signers

import (
	"math/big"
	"testing"

	gethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const testPrivateKey = "0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"

func TestLocalSignerImplementsPublicSigner(t *testing.T) {
	var _ Signer = (*LocalSigner)(nil)

	signer, err := NewLocalSigner(testPrivateKey, 137)
	if err != nil {
		t.Fatal(err)
	}
	if signer.Address() == "" {
		t.Fatal("expected signer address")
	}
	if signer.ChainID() != 137 {
		t.Fatalf("chainID=%d", signer.ChainID())
	}

	var hash [32]byte
	copy(hash[:], []byte("fixture-hash-32-byte-value-0000"))
	sig, err := signer.SignHash(hash)
	if err != nil {
		t.Fatalf("sign hash: %v", err)
	}
	if len(sig) != 65 {
		t.Fatalf("hash signature length=%d want 65", len(sig))
	}

	typedSig, err := signer.SignEIP712(minimalTypedData())
	if err != nil {
		t.Fatalf("sign eip712: %v", err)
	}
	if len(typedSig) != 65 {
		t.Fatalf("typed signature length=%d want 65", len(typedSig))
	}
}

func TestNewLocalSignerRejectsBadKeyWithoutLeakingIt(t *testing.T) {
	_, err := NewLocal("0xnot-a-private-key", 137)
	if err == nil {
		t.Fatal("expected invalid key error")
	}
	if got := RedactSecret("0xnot-a-private-key"); got == "0xnot-a-private-key" {
		t.Fatalf("secret was not redacted: %q", got)
	}
}

func minimalTypedData() apitypes.TypedData {
	return apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
			"Message": []apitypes.Type{
				{Name: "contents", Type: "string"},
			},
		},
		PrimaryType: "Message",
		Domain: apitypes.TypedDataDomain{
			Name:    "polygolem-test",
			Version: "1",
			ChainId: (*gethmath.HexOrDecimal256)(big.NewInt(137)),
		},
		Message: apitypes.TypedDataMessage{
			"contents": "hello",
		},
	}
}
