package authclobprobe

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/clob"
)

const testPrivateKey = "0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"

func TestRunnerUsesConfiguredCredentialsForReadOnlyProbe(t *testing.T) {
	client := &fakeCLOB{
		orders:  []clob.OrderRecord{{ID: "0xorder"}, {ID: "0xorder2"}},
		trades:  []clob.TradeRecord{{ID: "trade-1"}},
		balance: &clob.BalanceAllowanceResponse{Balance: "1000000", Allowance: "999"},
	}
	runner := New(Config{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		L2Credentials: func() (auth.APIKey, bool) {
			return auth.APIKey{Key: "configured-key", Secret: "secret", Passphrase: "pass"}, true
		},
		CLOB: client,
	})

	result, err := runner.Probe(context.Background())
	if err != nil {
		t.Fatalf("Probe returned error: %v", err)
	}
	if result.CredentialSource != "configured_clob_l2" || !result.ReadOnly || result.DeriveAPIKeyCalled {
		t.Fatalf("unexpected metadata: %+v", result)
	}
	if !strings.EqualFold(result.EOAAddress, "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23") {
		t.Fatalf("eoaAddress=%q", result.EOAAddress)
	}
	if !strings.EqualFold(result.DepositWallet, "0xfd5041047be8c192c725a66228f141196fa3cf9c") {
		t.Fatalf("depositWallet=%q", result.DepositWallet)
	}
	if client.setKey.Key != "configured-key" {
		t.Fatalf("set key=%+v", client.setKey)
	}
	if client.ordersPrivateKey != testPrivateKey || client.tradesPrivateKey != testPrivateKey || client.balancePrivateKey != testPrivateKey {
		t.Fatalf("private keys orders=%q trades=%q balance=%q", client.ordersPrivateKey, client.tradesPrivateKey, client.balancePrivateKey)
	}
	if client.balanceParams.AssetType != "COLLATERAL" {
		t.Fatalf("balance asset type=%q", client.balanceParams.AssetType)
	}
	if result.Orders.Count != 2 || result.Trades.Count != 1 || result.BalanceAllowance.Balance != "1000000" || result.BalanceAllowance.Allowance != "999" {
		t.Fatalf("unexpected result=%+v", result)
	}
}

func TestRunnerRejectsMissingOrInvalidConfiguredCredentialsBeforeReads(t *testing.T) {
	client := &fakeCLOB{}
	runner := New(Config{
		PrivateKey:    func() (string, error) { return testPrivateKey, nil },
		L2Credentials: func() (auth.APIKey, bool) { return auth.APIKey{}, false },
		CLOB:          client,
	})
	_, err := runner.Probe(context.Background())
	if err == nil || !strings.Contains(err.Error(), "configured CLOB L2 credentials are required") {
		t.Fatalf("missing credentials error=%v", err)
	}
	if client.setCalled || client.ordersCalled || client.tradesCalled || client.balanceCalled {
		t.Fatalf("client was used despite missing credentials: %+v", client)
	}

	runner = New(Config{
		PrivateKey:    func() (string, error) { return testPrivateKey, nil },
		L2Credentials: func() (auth.APIKey, bool) { return auth.APIKey{Key: "configured-key"}, true },
		CLOB:          client,
	})
	_, err = runner.Probe(context.Background())
	if err == nil || !strings.Contains(err.Error(), "configured CLOB L2 credentials invalid") {
		t.Fatalf("invalid credentials error=%v", err)
	}
}

func TestRunnerPropagatesPrivateKeyAndReadErrors(t *testing.T) {
	wantKeyErr := errors.New("POLYMARKET_PRIVATE_KEY is required")
	runner := New(Config{PrivateKey: func() (string, error) { return "", wantKeyErr }})
	_, err := runner.Probe(context.Background())
	if !errors.Is(err, wantKeyErr) {
		t.Fatalf("private key error=%v, want %v", err, wantKeyErr)
	}

	wantReadErr := errors.New("clob down")
	runner = New(Config{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		L2Credentials: func() (auth.APIKey, bool) {
			return auth.APIKey{Key: "configured-key", Secret: "secret", Passphrase: "pass"}, true
		},
		CLOB: &fakeCLOB{ordersErr: wantReadErr},
	})
	_, err = runner.Probe(context.Background())
	if !errors.Is(err, wantReadErr) || !strings.Contains(err.Error(), "read CLOB orders") {
		t.Fatalf("read error=%v, want wrapped %v", err, wantReadErr)
	}
}

type fakeCLOB struct {
	setKey    auth.APIKey
	setCalled bool

	orders           []clob.OrderRecord
	ordersErr        error
	ordersCalled     bool
	ordersPrivateKey string

	trades           []clob.TradeRecord
	tradesErr        error
	tradesCalled     bool
	tradesPrivateKey string

	balance           *clob.BalanceAllowanceResponse
	balanceErr        error
	balanceCalled     bool
	balancePrivateKey string
	balanceParams     clob.BalanceAllowanceParams
}

func (f *fakeCLOB) SetL2Credentials(key auth.APIKey) {
	f.setCalled = true
	f.setKey = key
}

func (f *fakeCLOB) ListOrders(_ context.Context, privateKey string) ([]clob.OrderRecord, error) {
	f.ordersCalled = true
	f.ordersPrivateKey = privateKey
	return f.orders, f.ordersErr
}

func (f *fakeCLOB) ListTrades(_ context.Context, privateKey string) ([]clob.TradeRecord, error) {
	f.tradesCalled = true
	f.tradesPrivateKey = privateKey
	return f.trades, f.tradesErr
}

func (f *fakeCLOB) BalanceAllowance(_ context.Context, privateKey string, params clob.BalanceAllowanceParams) (*clob.BalanceAllowanceResponse, error) {
	f.balanceCalled = true
	f.balancePrivateKey = privateKey
	f.balanceParams = params
	return f.balance, f.balanceErr
}
