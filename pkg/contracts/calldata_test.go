package contracts

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// keccakSelector derives the canonical 4-byte selector so the embedded ABI
// JSON cannot silently drift from the real method signatures.
func keccakSelector(signature string) []byte {
	return crypto.Keccak256([]byte(signature))[:4]
}

func TestCalldataSelectorsMatchCanonicalSignatures(t *testing.T) {
	spender := "0x1111111111111111111111111111111111111111"
	owner := "0x2222222222222222222222222222222222222222"
	amount := big.NewInt(5)

	cases := []struct {
		name      string
		signature string
		build     func() ([]byte, error)
	}{
		{"approve", "approve(address,uint256)", func() ([]byte, error) { return ERC20ApproveCalldata(spender, amount) }},
		{"transfer", "transfer(address,uint256)", func() ([]byte, error) { return ERC20TransferCalldata(spender, amount) }},
		{"allowance", "allowance(address,address)", func() ([]byte, error) { return ERC20AllowanceCalldata(owner, spender) }},
		{"balanceOf", "balanceOf(address)", func() ([]byte, error) { return ERC20BalanceOfCalldata(owner) }},
		{"setApprovalForAll", "setApprovalForAll(address,bool)", func() ([]byte, error) { return ERC1155SetApprovalForAllCalldata(spender, true) }},
		{"isApprovedForAll", "isApprovedForAll(address,address)", func() ([]byte, error) { return ERC1155IsApprovedForAllCalldata(owner, spender) }},
		{"wrap", "wrap(address,address,uint256)", func() ([]byte, error) { return RampWrapCalldata(USDCE, owner, amount) }},
		{"unwrap", "unwrap(address,address,uint256)", func() ([]byte, error) { return RampUnwrapCalldata(USDCE, owner, amount) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.build()
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			want := keccakSelector(tc.signature)
			if !bytes.Equal(data[:4], want) {
				t.Fatalf("selector = %x; want %x for %s", data[:4], want, tc.signature)
			}
		})
	}
}

// The relayer's deposit-wallet batches hand-roll the same approve /
// setApprovalForAll encodings; the exported builders must produce
// byte-identical calldata for the max-uint approval case.
func TestCalldataMatchesRelayerHandRolledEncoding(t *testing.T) {
	spender := strings.ToLower(strings.TrimPrefix(CTFExchangeV2, "0x"))

	approve, err := ERC20ApproveCalldata(CTFExchangeV2, nil)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	wantApprove := "095ea7b3" +
		strings.Repeat("0", 64-len(spender)) + spender +
		strings.Repeat("f", 64)
	if got := hex.EncodeToString(approve); got != wantApprove {
		t.Fatalf("approve calldata = %s; want %s", got, wantApprove)
	}

	setAll, err := ERC1155SetApprovalForAllCalldata(CTFExchangeV2, true)
	if err != nil {
		t.Fatalf("setApprovalForAll: %v", err)
	}
	wantSetAll := "a22cb465" +
		strings.Repeat("0", 64-len(spender)) + spender +
		strings.Repeat("0", 63) + "1"
	if got := hex.EncodeToString(setAll); got != wantSetAll {
		t.Fatalf("setApprovalForAll calldata = %s; want %s", got, wantSetAll)
	}
}

func TestERC20ApproveCalldataNilAmountIsMaxUint(t *testing.T) {
	data, err := ERC20ApproveCalldata("0x1111111111111111111111111111111111111111", nil)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	amountWord := data[len(data)-32:]
	if got := new(big.Int).SetBytes(amountWord); got.Cmp(MaxUint256()) != 0 {
		t.Fatalf("nil amount encoded %s; want MaxUint256", got)
	}
}

func TestRampCalldataRejectsNonPositiveAmounts(t *testing.T) {
	for _, amount := range []*big.Int{nil, big.NewInt(0), big.NewInt(-1)} {
		if _, err := RampWrapCalldata(USDCE, PUSD, amount); err == nil {
			t.Fatalf("wrap accepted amount %v", amount)
		}
		if _, err := RampUnwrapCalldata(USDCE, PUSD, amount); err == nil {
			t.Fatalf("unwrap accepted amount %v", amount)
		}
	}
}

func TestDecodeUint256Result(t *testing.T) {
	word := make([]byte, 32)
	word[31] = 42
	got, err := DecodeUint256Result(word)
	if err != nil || got.Int64() != 42 {
		t.Fatalf("DecodeUint256Result = %v, %v; want 42, nil", got, err)
	}
	if _, err := DecodeUint256Result(word[:31]); err == nil {
		t.Fatal("short word accepted")
	}
	if _, err := DecodeUint256Result(nil); err == nil {
		t.Fatal("empty return accepted")
	}
}

func TestDecodeBoolResult(t *testing.T) {
	word := make([]byte, 32)
	if got, err := DecodeBoolResult(word); err != nil || got {
		t.Fatalf("zero word = %v, %v; want false, nil", got, err)
	}
	word[31] = 1
	if got, err := DecodeBoolResult(word); err != nil || !got {
		t.Fatalf("one word = %v, %v; want true, nil", got, err)
	}
	word[31] = 2
	if _, err := DecodeBoolResult(word); err == nil {
		t.Fatal("word 2 accepted as bool")
	}
	if _, err := DecodeBoolResult(word[:31]); err == nil {
		t.Fatal("short word accepted")
	}
}

func TestMaxUint256Is2To256Minus1(t *testing.T) {
	want := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))
	if MaxUint256().Cmp(want) != 0 {
		t.Fatalf("MaxUint256 = %s", MaxUint256())
	}
}
