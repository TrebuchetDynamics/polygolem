package authstatus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/clob"
)

const testPrivateKey = "0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"

func TestRunnerStatusDerivesWalletAndReportsDeployNextStep(t *testing.T) {
	relayer := &fakeRelayer{deployed: false}
	clobClient := &fakeCLOB{deriveKey: auth.APIKey{Key: "eoa-key", Secret: "secret", Passphrase: "pass"}}
	runner := New(Config{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		Relayer:    func(context.Context) (RelayerReader, error) { return relayer, nil },
		CLOB:       clobClient,
	})

	status, err := runner.Status(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if !strings.EqualFold(status.EOAAddress, "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23") {
		t.Fatalf("eoaAddress=%q", status.EOAAddress)
	}
	if !strings.EqualFold(status.DepositWallet, "0xfd5041047be8c192c725a66228f141196fa3cf9c") {
		t.Fatalf("depositWallet=%q", status.DepositWallet)
	}
	if relayer.owner != status.EOAAddress {
		t.Fatalf("relayer owner=%q want %q", relayer.owner, status.EOAAddress)
	}
	if clobClient.derivePrivateKey != testPrivateKey {
		t.Fatalf("derive private key=%q", clobClient.derivePrivateKey)
	}
	if status.DepositWalletDeployed || !status.EOAAPIKeyExists || status.DepositWalletAPIKeyExists || status.CanTrade {
		t.Fatalf("unexpected readiness status=%+v", status)
	}
	if !strings.Contains(status.NextStep, "deposit-wallet deploy --wait") || status.Help == "" {
		t.Fatalf("nextStep/help missing: %+v", status)
	}
}

func TestRunnerStatusUsesConfiguredDepositKeyWithoutLiveOrderProbe(t *testing.T) {
	clobClient := &fakeCLOB{deriveErr: errors.New("no eoa key")}
	validKey := auth.APIKey{Key: "configured-key", Secret: "secret", Passphrase: "pass"}
	runner := New(Config{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		Relayer:    func(context.Context) (RelayerReader, error) { return &fakeRelayer{deployed: true}, nil },
		CLOB:       clobClient,
		L2Credentials: func() (auth.APIKey, bool) {
			return validKey, true
		},
	})

	status, err := runner.Status(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.EOAAPIKeyExists || !status.DepositWalletAPIKeyExists || !status.CanTrade || status.NextStep != "" || status.Help != "" {
		t.Fatalf("unexpected readiness status=%+v", status)
	}
	if clobClient.setKey.Key != "configured-key" {
		t.Fatalf("configured key not set: %+v", clobClient.setKey)
	}
	if clobClient.listOrdersCalled {
		t.Fatal("ListOrders called without CheckDepositKey")
	}
}

func TestRunnerStatusCheckDepositKeyUsesLiveOrderProbeWhenDeployed(t *testing.T) {
	clobClient := &fakeCLOB{deriveErr: errors.New("no eoa key"), orders: []clob.OrderRecord{{ID: "0xorder"}}}
	runner := New(Config{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		Relayer:    func(context.Context) (RelayerReader, error) { return &fakeRelayer{deployed: true}, nil },
		CLOB:       clobClient,
	})

	status, err := runner.Status(context.Background(), Request{CheckDepositKey: true})
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if !clobClient.listOrdersCalled || clobClient.listOrdersPrivateKey != testPrivateKey {
		t.Fatalf("ListOrders called=%v privateKey=%q", clobClient.listOrdersCalled, clobClient.listOrdersPrivateKey)
	}
	if !status.DepositWalletAPIKeyExists || !status.CanTrade {
		t.Fatalf("deposit key live probe did not mark canTrade: %+v", status)
	}
}

func TestRunnerStatusPropagatesPrivateKeyError(t *testing.T) {
	want := errors.New("POLYMARKET_PRIVATE_KEY is required")
	runner := New(Config{PrivateKey: func() (string, error) { return "", want }})
	_, err := runner.Status(context.Background(), Request{})
	if !errors.Is(err, want) {
		t.Fatalf("error=%v, want %v", err, want)
	}
}

type fakeRelayer struct {
	deployed bool
	err      error
	owner    string
}

func (f *fakeRelayer) IsDeployed(_ context.Context, owner string) (bool, error) {
	f.owner = owner
	return f.deployed, f.err
}

type fakeCLOB struct {
	deriveKey            auth.APIKey
	deriveErr            error
	derivePrivateKey     string
	setKey               auth.APIKey
	orders               []clob.OrderRecord
	ordersErr            error
	listOrdersCalled     bool
	listOrdersPrivateKey string
}

func (f *fakeCLOB) DeriveAPIKey(_ context.Context, privateKey string) (auth.APIKey, error) {
	f.derivePrivateKey = privateKey
	return f.deriveKey, f.deriveErr
}

func (f *fakeCLOB) SetL2Credentials(key auth.APIKey) {
	f.setKey = key
}

func (f *fakeCLOB) ListOrders(_ context.Context, privateKey string) ([]clob.OrderRecord, error) {
	f.listOrdersCalled = true
	f.listOrdersPrivateKey = privateKey
	return f.orders, f.ordersErr
}
