package depositwalletreads

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/relayer"
	"github.com/TrebuchetDynamics/polygolem/pkg/contracts"
)

const testPrivateKey = "0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"

func TestRunnerDerivesDepositWalletAndReadsNonce(t *testing.T) {
	fake := &fakeRelayer{nonce: "7"}
	var relayerPrivateKey string
	runner := New(Config{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		Relayer: func(_ context.Context, privateKey string) (RelayerReader, error) {
			relayerPrivateKey = privateKey
			return fake, nil
		},
	})

	derived, err := runner.Derive(context.Background())
	if err != nil {
		t.Fatalf("Derive returned error: %v", err)
	}
	if !strings.EqualFold(derived.Owner, "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23") {
		t.Fatalf("owner=%q", derived.Owner)
	}
	if !strings.EqualFold(derived.DepositWallet, "0xfd5041047be8c192c725a66228f141196fa3cf9c") {
		t.Fatalf("depositWallet=%q", derived.DepositWallet)
	}

	nonce, err := runner.Nonce(context.Background())
	if err != nil {
		t.Fatalf("Nonce returned error: %v", err)
	}
	if nonce.Address != derived.Owner || nonce.Type != "WALLET" || nonce.Nonce != "7" {
		t.Fatalf("nonce=%+v derived=%+v", nonce, derived)
	}
	if relayerPrivateKey != testPrivateKey {
		t.Fatalf("relayer private key=%q", relayerPrivateKey)
	}
	if fake.nonceOwner != derived.Owner {
		t.Fatalf("nonce owner=%q want %q", fake.nonceOwner, derived.Owner)
	}
}

func TestRunnerStatusUsesOnchainCodeWhenRelayerReportsNotDeployed(t *testing.T) {
	fake := &fakeRelayer{deployed: false, nonce: "9"}
	var checkedWallet string
	var checkedRPCURL string
	runner := New(Config{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		Relayer:    func(context.Context, string) (RelayerReader, error) { return fake, nil },
		Deployment: func(_ context.Context, wallet string, rpcURL string) (contracts.DeploymentStatus, error) {
			checkedWallet = wallet
			checkedRPCURL = rpcURL
			return contracts.DeploymentStatus{Address: wallet, Deployed: true, Source: "test"}, nil
		},
		RPCURL: func() string { return "https://rpc.example" },
	})

	status, err := runner.Status(context.Background(), StatusRequest{})
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if !status.Deployed || status.RelayerDeployed || !status.OnchainCodeDeployed {
		t.Fatalf("deployment fields=%+v", status)
	}
	if status.DeploymentStatusSource != "polygon_code" {
		t.Fatalf("deploymentStatusSource=%q", status.DeploymentStatusSource)
	}
	if status.WalletNonce != "9" {
		t.Fatalf("walletNonce=%q", status.WalletNonce)
	}
	if checkedWallet != status.DepositWallet || checkedRPCURL != "https://rpc.example" {
		t.Fatalf("deployment check wallet=%q rpcURL=%q status=%+v", checkedWallet, checkedRPCURL, status)
	}
}

func TestRunnerStatusCapturesNonceErrorsAndOptionalEnableTrading(t *testing.T) {
	wantNonceErr := errors.New("relayer timeout")
	validationCalled := false
	runner := New(Config{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		Relayer: func(context.Context, string) (RelayerReader, error) {
			return &fakeRelayer{deployed: true, nonceErr: wantNonceErr}, nil
		},
		EnableTrading: func(_ context.Context, privateKey string, wallet string, nonce string) (map[string]interface{}, error) {
			validationCalled = true
			if privateKey != testPrivateKey || !strings.HasPrefix(nonce, "error: relayer timeout") || wallet == "" {
				t.Fatalf("enable trading args privateKey=%q wallet=%q nonce=%q", privateKey, wallet, nonce)
			}
			return map[string]interface{}{"ready": true}, nil
		},
	})

	status, err := runner.Status(context.Background(), StatusRequest{CheckEnableTrading: true})
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.WalletNonce != "error: relayer timeout" {
		t.Fatalf("walletNonce=%q", status.WalletNonce)
	}
	if ready, _ := status.EnableTrading["ready"].(bool); !ready || !validationCalled {
		t.Fatalf("enableTrading=%+v validationCalled=%v", status.EnableTrading, validationCalled)
	}
}

func TestRunnerTransactionUsesStatusRelayerAndValidatesSigner(t *testing.T) {
	fake := &fakeRelayer{tx: &relayer.RelayerTransaction{TransactionID: "tx-1", State: "MINED"}}
	runner := New(Config{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		Relayer:    func(context.Context, string) (RelayerReader, error) { return fake, nil },
	})

	tx, err := runner.Transaction(context.Background(), "tx-1")
	if err != nil {
		t.Fatalf("Transaction returned error: %v", err)
	}
	if tx.TransactionID != "tx-1" || fake.txID != "tx-1" {
		t.Fatalf("tx=%+v fake.txID=%q", tx, fake.txID)
	}
}

type fakeRelayer struct {
	nonce    string
	nonceErr error

	deployed    bool
	deployErr   error
	deployOwner string

	tx   *relayer.RelayerTransaction
	txID string

	nonceOwner string
}

func (f *fakeRelayer) GetNonce(_ context.Context, owner string) (string, error) {
	f.nonceOwner = owner
	if f.nonceErr != nil {
		return "", f.nonceErr
	}
	return f.nonce, nil
}

func (f *fakeRelayer) IsDeployed(_ context.Context, owner string) (bool, error) {
	f.deployOwner = owner
	return f.deployed, f.deployErr
}

func (f *fakeRelayer) GetTransaction(_ context.Context, txID string) (*relayer.RelayerTransaction, error) {
	f.txID = txID
	return f.tx, nil
}
