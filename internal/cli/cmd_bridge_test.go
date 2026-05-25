package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/pkg/bridge"
	"github.com/spf13/cobra"
)

type fakeBridgeAssetsRunner struct {
	called bool
	result *bridge.SupportedAssetsResponse
}

func (f *fakeBridgeAssetsRunner) Run(context.Context) (*bridge.SupportedAssetsResponse, error) {
	f.called = true
	return f.result, nil
}

type fakeBridgeClient struct {
	depositAddressArg string
	depositResult     *bridge.CreateDepositAddressResponse
	statusAddressArg  string
	statusResult      *bridge.DepositStatusResponse
	quoteRequest      bridge.QuoteRequest
	quoteResult       *bridge.QuoteResponse
}

func (f *fakeBridgeClient) CreateDepositAddress(_ context.Context, address string) (*bridge.CreateDepositAddressResponse, error) {
	f.depositAddressArg = address
	return f.depositResult, nil
}

func (f *fakeBridgeClient) GetDepositStatus(_ context.Context, address string) (*bridge.DepositStatusResponse, error) {
	f.statusAddressArg = address
	return f.statusResult, nil
}

func (f *fakeBridgeClient) GetQuote(_ context.Context, req bridge.QuoteRequest) (*bridge.QuoteResponse, error) {
	f.quoteRequest = req
	return f.quoteResult, nil
}

func newBridgeTestRoot(assets bridgeAssetsRunner, bc bridgeClient, args ...string) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	root := &cobra.Command{Use: "polygolem", SilenceUsage: true, SilenceErrors: true}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.PersistentFlags().Bool("json", false, "emit JSON output")
	root.AddCommand(newBridgeCommand(assets, bc))
	root.SetArgs(args)
	installJSONContract(root)
	return root, &stdout, &stderr
}

func TestBridgeAssetsCommandDelegatesToRunnerAndJSONEnvelope(t *testing.T) {
	fake := &fakeBridgeAssetsRunner{result: &bridge.SupportedAssetsResponse{SupportedAssets: []bridge.SupportedAsset{{
		ChainID:   "137",
		ChainName: "Polygon",
		Token:     bridge.TokenInfo{Symbol: "USDC", Decimals: 6},
	}}}}
	root, stdout, stderr := newBridgeTestRoot(fake, &fakeBridgeClient{}, "--json", "bridge", "assets")

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr.String())
	}
	if !fake.called {
		t.Fatal("bridge assets runner was not called")
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "bridge assets" {
		t.Fatalf("meta.command=%q, want bridge assets", got.Meta.Command)
	}
	var data bridge.SupportedAssetsResponse
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("data is not bridge assets payload: %v\n%s", err, got.Data)
	}
	if len(data.SupportedAssets) != 1 || data.SupportedAssets[0].ChainID != "137" || data.SupportedAssets[0].Token.Symbol != "USDC" {
		t.Fatalf("data=%+v", data)
	}
}

func TestBridgeStatusCommandDelegatesToClientAndJSONEnvelope(t *testing.T) {
	bc := &fakeBridgeClient{statusResult: &bridge.DepositStatusResponse{Transactions: []bridge.DepositTransaction{{
		FromChainID: "1",
		ToChainID:   "137",
		Status:      "confirmed",
	}}}}
	root, stdout, stderr := newBridgeTestRoot(&fakeBridgeAssetsRunner{}, bc, "--json", "bridge", "status", "0xdeposit")

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr.String())
	}
	if bc.statusAddressArg != "0xdeposit" {
		t.Fatalf("status address=%q, want 0xdeposit", bc.statusAddressArg)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "bridge status" {
		t.Fatalf("meta.command=%q, want bridge status", got.Meta.Command)
	}
	var data bridge.DepositStatusResponse
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("data is not bridge status payload: %v\n%s", err, got.Data)
	}
	if len(data.Transactions) != 1 || data.Transactions[0].Status != "confirmed" {
		t.Fatalf("data=%+v", data)
	}
}

func TestBridgeQuoteCommandDelegatesToClientAndJSONEnvelope(t *testing.T) {
	bc := &fakeBridgeClient{quoteResult: &bridge.QuoteResponse{
		EstCheckoutTimeMs:  120000,
		EstInputUsd:        100,
		EstOutputUsd:       99.4,
		EstToTokenBaseUnit: "99400000",
		QuoteID:            "quote-1",
	}}
	root, stdout, stderr := newBridgeTestRoot(
		&fakeBridgeAssetsRunner{},
		bc,
		"--json", "bridge", "quote",
		"--from-amount-base-unit", "1000000",
		"--from-chain-id", "1",
		"--from-token-address", "0xa0b8",
		"--recipient-address", "0xrecipient",
		"--to-chain-id", "137",
		"--to-token-address", "0x2791",
	)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr.String())
	}
	if bc.quoteRequest.FromAmountBaseUnit != "1000000" || bc.quoteRequest.RecipientAddress != "0xrecipient" {
		t.Fatalf("quote request=%+v", bc.quoteRequest)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "bridge quote" {
		t.Fatalf("meta.command=%q, want bridge quote", got.Meta.Command)
	}
	var data bridge.QuoteResponse
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("data is not bridge quote payload: %v\n%s", err, got.Data)
	}
	if data.QuoteID != "quote-1" || data.EstToTokenBaseUnit != "99400000" {
		t.Fatalf("data=%+v", data)
	}
}
