package depositwalletfunding

import (
	"context"
	"math/big"
	"testing"
)

const testPrivateKey = "0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"

func TestFundDerivesWalletParsesAmountAndTransfers(t *testing.T) {
	var gotKey, gotWallet, gotRPC string
	var gotAmount *big.Int
	result, err := Fund(context.Background(), FundConfig{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		TransferPUSD: func(ctx context.Context, privateKeyHex, toAddress string, amount *big.Int, rpcURL string) (string, error) {
			gotKey = privateKeyHex
			gotWallet = toAddress
			gotAmount = new(big.Int).Set(amount)
			gotRPC = rpcURL
			return "0xtx", nil
		},
	}, FundRequest{AmountPUSD: "0.710000", RPCURL: "https://rpc.example"})
	if err != nil {
		t.Fatalf("Fund returned error: %v", err)
	}
	if result.TxHash != "0xtx" || result.From == "" || result.To == "" || result.Amount != "0.710000" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if gotKey != testPrivateKey || gotWallet != result.To || gotRPC != "https://rpc.example" {
		t.Fatalf("transfer args key=%q wallet=%q rpc=%q result=%+v", gotKey, gotWallet, gotRPC, result)
	}
	if gotAmount.String() != "710000" {
		t.Fatalf("amount=%s, want 710000", gotAmount)
	}
}

func TestFundRejectsMissingAndZeroAmountBeforeTransfer(t *testing.T) {
	called := false
	_, err := Fund(context.Background(), FundConfig{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		TransferPUSD: func(ctx context.Context, privateKeyHex, toAddress string, amount *big.Int, rpcURL string) (string, error) {
			called = true
			return "", nil
		},
	}, FundRequest{AmountPUSD: "0"})
	if err == nil {
		t.Fatal("expected zero amount error")
	}
	if called {
		t.Fatal("transfer was called for invalid amount")
	}
}
