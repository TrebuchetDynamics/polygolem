package depositwalletsettlement

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/pkg/data"
	"github.com/TrebuchetDynamics/polygolem/pkg/settlement"
)

const testPrivateKey = "0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318"

func TestRunnerSettlementStatusDerivesWalletAndPassesReadinessInputs(t *testing.T) {
	client := data.NewClient(data.Config{BaseURL: "https://data.example"})
	var gotOwner string
	var gotWallet string
	var gotOptions settlement.ReadinessOptions
	var gotClient *data.Client
	runner := New(Config{
		PrivateKey:        func() (string, error) { return testPrivateKey, nil },
		DataClient:        func() *data.Client { return client },
		RelayerConfigured: func() bool { return false },
		RPCURL:            func() string { return "https://env-rpc.example" },
		CheckReadiness: func(_ context.Context, dataClient *data.Client, owner string, wallet string, opts settlement.ReadinessOptions) (*settlement.Readiness, error) {
			gotClient = dataClient
			gotOwner = owner
			gotWallet = wallet
			gotOptions = opts
			return &settlement.Readiness{Status: settlement.StatusMissingRelayerCreds, Owner: owner, DepositWallet: wallet, RelayerConfigured: opts.RelayerConfigured}, nil
		},
	})

	status, err := runner.SettlementStatus(context.Background(), StatusRequest{RPCURL: "https://flag-rpc.example"})
	if err != nil {
		t.Fatalf("SettlementStatus returned error: %v", err)
	}
	if !strings.EqualFold(gotOwner, "0x2c7536E3605D9C16a7a3D7b1898e529396a65c23") {
		t.Fatalf("owner=%q", gotOwner)
	}
	if !strings.EqualFold(gotWallet, "0xd2C50736787e5eeefA6c2E81496AE56d51D6b7B1") {
		t.Fatalf("wallet=%q", gotWallet)
	}
	if status.DepositWallet != gotWallet || status.Status != settlement.StatusMissingRelayerCreds {
		t.Fatalf("status=%+v gotWallet=%q", status, gotWallet)
	}
	if gotClient != client {
		t.Fatal("readiness did not receive configured data client")
	}
	if gotOptions.RPCURL != "https://flag-rpc.example" {
		t.Fatalf("rpcURL=%q", gotOptions.RPCURL)
	}
	if gotOptions.RelayerConfigured {
		t.Fatal("relayerConfigured=true, want false")
	}
}

func TestRunnerRedeemableUsesDepositWalletAsDataAPIUser(t *testing.T) {
	client := data.NewClient(data.Config{BaseURL: "https://data.example"})
	var gotClient *data.Client
	var gotWallet string
	runner := New(Config{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		DataClient: func() *data.Client { return client },
		FindRedeemable: func(_ context.Context, dataClient *data.Client, wallet string) ([]settlement.RedeemablePosition, error) {
			gotClient = dataClient
			gotWallet = wallet
			return []settlement.RedeemablePosition{{TokenID: "token-1", ConditionID: "0xcondition", Outcome: "Up"}}, nil
		},
	})

	result, err := runner.Redeemable(context.Background())
	if err != nil {
		t.Fatalf("Redeemable returned error: %v", err)
	}
	if gotClient != client {
		t.Fatal("finder did not receive configured data client")
	}
	if !strings.EqualFold(gotWallet, result.DepositWallet) || !strings.EqualFold(result.DepositWallet, "0xd2C50736787e5eeefA6c2E81496AE56d51D6b7B1") {
		t.Fatalf("wallet got=%q result=%q", gotWallet, result.DepositWallet)
	}
	if result.Count != 1 || len(result.Positions) != 1 || result.Positions[0].TokenID != "token-1" {
		t.Fatalf("result=%+v", result)
	}
}

func TestRunnerRedeemablePropagatesPrivateKeyAndFinderErrors(t *testing.T) {
	wantKeyErr := errors.New("missing private key")
	runner := New(Config{PrivateKey: func() (string, error) { return "", wantKeyErr }})
	_, err := runner.Redeemable(context.Background())
	if !errors.Is(err, wantKeyErr) {
		t.Fatalf("private key error=%v, want %v", err, wantKeyErr)
	}

	wantFinderErr := errors.New("data unavailable")
	runner = New(Config{
		PrivateKey: func() (string, error) { return testPrivateKey, nil },
		FindRedeemable: func(context.Context, *data.Client, string) ([]settlement.RedeemablePosition, error) {
			return nil, wantFinderErr
		},
	})
	_, err = runner.Redeemable(context.Background())
	if !errors.Is(err, wantFinderErr) || !strings.Contains(err.Error(), "find redeemable") {
		t.Fatalf("finder error=%v, want wrapped %v", err, wantFinderErr)
	}
}
