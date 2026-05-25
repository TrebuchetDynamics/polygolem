package clobbalances

import (
	"context"
	"errors"
	"reflect"
	"testing"

	internalclob "github.com/TrebuchetDynamics/polygolem/internal/clob"
)

func TestRunnerBalanceValidatesOutputBeforeLoadingPrivateKey(t *testing.T) {
	privateKeyCalled := false
	runner := New(Config{
		Reader: &fakeReader{},
		PrivateKey: func() (string, error) {
			privateKeyCalled = true
			return "0xprivate", nil
		},
	})

	_, err := runner.Balance(context.Background(), Request{Output: "table"})
	if err == nil || err.Error() != "only --output json is supported" {
		t.Fatalf("error=%v, want only --output json is supported", err)
	}
	if privateKeyCalled {
		t.Fatal("private key loader was called before output validation")
	}
}

func TestRunnerReadsAndNormalizesBalanceAndUpdateBalance(t *testing.T) {
	reader := &fakeReader{
		balance: &internalclob.BalanceAllowanceResponse{
			Balance: "14000000",
			Allowances: map[string]string{
				"0xspender": "1000000",
			},
		},
		update: &internalclob.BalanceAllowanceResponse{
			Balance:   "2500000",
			Allowance: "500000",
		},
	}
	runner := New(Config{
		Reader: reader,
		PrivateKey: func() (string, error) {
			return "0xprivate", nil
		},
	})

	balance, err := runner.Balance(context.Background(), Request{AssetType: "collateral", TokenID: "token-1", Output: "json"})
	if err != nil {
		t.Fatalf("Balance returned error: %v", err)
	}
	if balance["balance"] != "14.000000" {
		t.Fatalf("balance=%v, want 14.000000", balance["balance"])
	}
	if !reflect.DeepEqual(balance["allowances"], reader.balance.Allowances) {
		t.Fatalf("allowances=%#v", balance["allowances"])
	}
	if reader.balancePrivateKey != "0xprivate" || reader.balanceParams.AssetType != "collateral" || reader.balanceParams.TokenID != "token-1" {
		t.Fatalf("balance privateKey=%q params=%+v", reader.balancePrivateKey, reader.balanceParams)
	}

	update, err := runner.UpdateBalance(context.Background(), Request{AssetType: "conditional", TokenID: "token-2"})
	if err != nil {
		t.Fatalf("UpdateBalance returned error: %v", err)
	}
	if update["balance"] != "2.500000" || update["allowance"] != "500000" {
		t.Fatalf("unexpected update response: %#v", update)
	}
	if reader.updatePrivateKey != "0xprivate" || reader.updateParams.AssetType != "conditional" || reader.updateParams.TokenID != "token-2" {
		t.Fatalf("update privateKey=%q params=%+v", reader.updatePrivateKey, reader.updateParams)
	}
}

func TestRunnerPropagatesPrivateKeyAndReaderErrors(t *testing.T) {
	wantKeyErr := errors.New("missing private key")
	runner := New(Config{Reader: &fakeReader{}, PrivateKey: func() (string, error) { return "", wantKeyErr }})
	_, err := runner.UpdateBalance(context.Background(), Request{})
	if !errors.Is(err, wantKeyErr) {
		t.Fatalf("private key error=%v, want %v", err, wantKeyErr)
	}

	wantReaderErr := errors.New("clob down")
	runner = New(Config{Reader: &fakeReader{err: wantReaderErr}, PrivateKey: func() (string, error) { return "0xprivate", nil }})
	_, err = runner.Balance(context.Background(), Request{})
	if !errors.Is(err, wantReaderErr) {
		t.Fatalf("reader error=%v, want %v", err, wantReaderErr)
	}
}

type fakeReader struct {
	balance *internalclob.BalanceAllowanceResponse
	update  *internalclob.BalanceAllowanceResponse
	err     error

	balancePrivateKey string
	balanceParams     internalclob.BalanceAllowanceParams
	updatePrivateKey  string
	updateParams      internalclob.BalanceAllowanceParams
}

func (f *fakeReader) BalanceAllowance(_ context.Context, privateKey string, params internalclob.BalanceAllowanceParams) (*internalclob.BalanceAllowanceResponse, error) {
	f.balancePrivateKey = privateKey
	f.balanceParams = params
	return f.balance, f.err
}

func (f *fakeReader) UpdateBalanceAllowance(_ context.Context, privateKey string, params internalclob.BalanceAllowanceParams) (*internalclob.BalanceAllowanceResponse, error) {
	f.updatePrivateKey = privateKey
	f.updateParams = params
	return f.update, f.err
}
