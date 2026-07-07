package depositwalletfunding

import (
	"context"
	"math/big"
	"testing"
)

func TestSwapParsesCapsAndSwapsToEOA(t *testing.T) {
	var gotKey, gotRPC string
	var gotOut, gotMax *big.Int
	result, err := Swap(context.Background(), SwapConfig{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		SwapPOLForExactPUSD: func(ctx context.Context, privateKeyHex string, amountOutPUSD, maxPOLInWei *big.Int, rpcURL string) (string, error) {
			gotKey = privateKeyHex
			gotOut = new(big.Int).Set(amountOutPUSD)
			gotMax = new(big.Int).Set(maxPOLInWei)
			gotRPC = rpcURL
			return "0xswap", nil
		},
	}, SwapRequest{AmountPUSDOut: "1.25", MaxPOLIn: "0.5", RPCURL: "https://rpc.example"})
	if err != nil {
		t.Fatalf("Swap returned error: %v", err)
	}
	if result.TxHash != "0xswap" || result.Recipient == "" || result.AmountPUSDOut != "1.25" || result.MaxPOLIn != "0.5" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if gotKey != testPrivateKey || gotRPC != "https://rpc.example" {
		t.Fatalf("swap args key=%q rpc=%q", gotKey, gotRPC)
	}
	if gotOut.String() != "1250000" {
		t.Fatalf("out=%s, want 1250000", gotOut)
	}
	if gotMax.String() != "500000000000000000" {
		t.Fatalf("max=%s, want 500000000000000000", gotMax)
	}
}

func TestSwapRejectsMissingCapBeforeSwap(t *testing.T) {
	called := false
	_, err := Swap(context.Background(), SwapConfig{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		SwapPOLForExactPUSD: func(ctx context.Context, privateKeyHex string, amountOutPUSD, maxPOLInWei *big.Int, rpcURL string) (string, error) {
			called = true
			return "", nil
		},
	}, SwapRequest{AmountPUSDOut: "1"})
	if err == nil {
		t.Fatal("expected missing --max-pol-in error")
	}
	if called {
		t.Fatal("swap was called for invalid request")
	}
}

func TestParsePOLAmountConvertsToWei(t *testing.T) {
	got, err := ParsePOLAmount("10.25")
	if err != nil {
		t.Fatalf("ParsePOLAmount returned error: %v", err)
	}
	if got.String() != "10250000000000000000" {
		t.Fatalf("wei=%s, want 10250000000000000000", got)
	}
}
