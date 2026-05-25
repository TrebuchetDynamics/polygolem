package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	internalclob "github.com/TrebuchetDynamics/polygolem/internal/clob"
	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/clobaccountreads"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/clobbalances"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/clobdiagnostics"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/clobmarketdata"
	"github.com/spf13/cobra"
)

type fakeCLOBMarketDataRunner struct {
	marketsRequest clobmarketdata.MarketsRequest
}

func (f *fakeCLOBMarketDataRunner) Book(context.Context, clobmarketdata.TokenRequest) (*polytypes.OrderBook, error) {
	return &polytypes.OrderBook{}, nil
}

func (f *fakeCLOBMarketDataRunner) TickSize(context.Context, clobmarketdata.TokenRequest) (*polytypes.TickSize, error) {
	return &polytypes.TickSize{}, nil
}

func (f *fakeCLOBMarketDataRunner) PriceHistory(context.Context, clobmarketdata.PriceHistoryRequest) (*polytypes.PriceHistory, error) {
	return &polytypes.PriceHistory{}, nil
}

func (f *fakeCLOBMarketDataRunner) Market(context.Context, clobmarketdata.ConditionRequest) (*polytypes.CLOBMarket, error) {
	return &polytypes.CLOBMarket{}, nil
}

func (f *fakeCLOBMarketDataRunner) MarketByToken(context.Context, clobmarketdata.TokenRequest) (*polytypes.CLOBMarketByTokenResponse, error) {
	return &polytypes.CLOBMarketByTokenResponse{}, nil
}

func (f *fakeCLOBMarketDataRunner) Markets(_ context.Context, req clobmarketdata.MarketsRequest) (*polytypes.CLOBPaginatedMarkets, error) {
	f.marketsRequest = req
	return &polytypes.CLOBPaginatedMarkets{NextCursor: "next", Count: 1}, nil
}

func TestCLOBMarketDataBuilderDelegatesToRunnerAndJSONEnvelope(t *testing.T) {
	fake := &fakeCLOBMarketDataRunner{}
	clobCmd := commandGroup("clob", "CLOB market data")
	addCLOBMarketDataCommands(clobCmd, fake)

	root := &cobra.Command{Use: "polygolem", SilenceUsage: true, SilenceErrors: true}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.PersistentFlags().Bool("json", false, "emit JSON output")
	root.AddCommand(clobCmd)
	root.SetArgs([]string{"--json", "clob", "markets", "--cursor", "cursor-1"})
	installJSONContract(root)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr.String())
	}
	if fake.marketsRequest.Cursor != "cursor-1" || fake.marketsRequest.Output != "json" {
		t.Fatalf("request=%+v", fake.marketsRequest)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "clob markets" {
		t.Fatalf("meta.command=%q, want clob markets", got.Meta.Command)
	}
	var data polytypes.CLOBPaginatedMarkets
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("data is not markets payload: %v\n%s", err, got.Data)
	}
	if data.NextCursor != "next" || data.Count != 1 {
		t.Fatalf("data=%+v", data)
	}
}

type fakeCLOBBalanceRunner struct {
	balanceRequest clobbalances.Request
}

func (f *fakeCLOBBalanceRunner) Balance(_ context.Context, req clobbalances.Request) (map[string]interface{}, error) {
	f.balanceRequest = req
	return map[string]interface{}{"balance": "1", "asset_type": req.AssetType, "token_id": req.TokenID}, nil
}

func (f *fakeCLOBBalanceRunner) UpdateBalance(context.Context, clobbalances.Request) (map[string]interface{}, error) {
	return map[string]interface{}{"balance": "2"}, nil
}

type fakeCLOBAccountReadRunner struct {
	orderRequest clobaccountreads.OrderRequest
}

func (f *fakeCLOBAccountReadRunner) Orders(context.Context, clobaccountreads.Request) ([]internalclob.OrderRecord, error) {
	return []internalclob.OrderRecord{{ID: "order-1"}}, nil
}

func (f *fakeCLOBAccountReadRunner) Order(_ context.Context, req clobaccountreads.OrderRequest) (*internalclob.OrderRecord, error) {
	f.orderRequest = req
	return &internalclob.OrderRecord{ID: req.OrderID}, nil
}

func (f *fakeCLOBAccountReadRunner) Trades(context.Context, clobaccountreads.Request) ([]internalclob.TradeRecord, error) {
	return []internalclob.TradeRecord{{ID: "trade-1"}}, nil
}

type fakeCLOBDiagnosticRunner struct {
	probeRequest clobdiagnostics.ProbeRequest
}

func (f *fakeCLOBDiagnosticRunner) ListBuilderFeeKeys(context.Context, clobdiagnostics.Request) ([]internalclob.BuilderFeeKeyRecord, error) {
	return []internalclob.BuilderFeeKeyRecord{{Key: "builder-key-1"}}, nil
}

func (f *fakeCLOBDiagnosticRunner) MarketTradesProbe(_ context.Context, req clobdiagnostics.ProbeRequest) (*internalclob.MarketTradesProbeResult, error) {
	f.probeRequest = req
	return &internalclob.MarketTradesProbeResult{Classification: internalclob.ProbeMarketWide, RowCount: 2}, nil
}

func TestCLOBAuthenticatedReadBuilderDelegatesToRunnersAndJSONEnvelope(t *testing.T) {
	t.Run("balance", func(t *testing.T) {
		balances := &fakeCLOBBalanceRunner{}
		stdout, stderr, err := executeCLOBAuthHelperForTest([]string{"--json", "clob", "balance", "--asset-type", "conditional", "--token-id", "token-1"}, balances, &fakeCLOBAccountReadRunner{}, &fakeCLOBDiagnosticRunner{})
		if err != nil {
			t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr)
		}
		if balances.balanceRequest.AssetType != "conditional" || balances.balanceRequest.TokenID != "token-1" || balances.balanceRequest.Output != "json" {
			t.Fatalf("request=%+v", balances.balanceRequest)
		}
		got := parseJSONEnvelopeForTest(t, stdout)
		if got.Meta.Command != "clob balance" {
			t.Fatalf("meta.command=%q, want clob balance", got.Meta.Command)
		}
		var data map[string]string
		if err := json.Unmarshal(got.Data, &data); err != nil {
			t.Fatalf("data is not balance payload: %v\n%s", err, got.Data)
		}
		if data["balance"] != "1" || data["asset_type"] != "conditional" || data["token_id"] != "token-1" {
			t.Fatalf("data=%+v", data)
		}
	})

	t.Run("order", func(t *testing.T) {
		accounts := &fakeCLOBAccountReadRunner{}
		stdout, stderr, err := executeCLOBAuthHelperForTest([]string{"--json", "clob", "order", "order-1"}, &fakeCLOBBalanceRunner{}, accounts, &fakeCLOBDiagnosticRunner{})
		if err != nil {
			t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr)
		}
		if accounts.orderRequest.OrderID != "order-1" || accounts.orderRequest.Output != "json" {
			t.Fatalf("request=%+v", accounts.orderRequest)
		}
		got := parseJSONEnvelopeForTest(t, stdout)
		var data internalclob.OrderRecord
		if err := json.Unmarshal(got.Data, &data); err != nil {
			t.Fatalf("data is not order payload: %v\n%s", err, got.Data)
		}
		if data.ID != "order-1" {
			t.Fatalf("data=%+v", data)
		}
	})

	t.Run("diagnostic probe", func(t *testing.T) {
		diagnostics := &fakeCLOBDiagnosticRunner{}
		stdout, stderr, err := executeCLOBAuthHelperForTest([]string{"--json", "clob", "market-trades-probe", "--market", "0xmarket", "--asset-id", "asset-1", "--cursor", "cursor-1"}, &fakeCLOBBalanceRunner{}, &fakeCLOBAccountReadRunner{}, diagnostics)
		if err != nil {
			t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr)
		}
		if diagnostics.probeRequest.Market != "0xmarket" || diagnostics.probeRequest.AssetID != "asset-1" || diagnostics.probeRequest.NextCursor != "cursor-1" || diagnostics.probeRequest.Output != "json" {
			t.Fatalf("request=%+v", diagnostics.probeRequest)
		}
		got := parseJSONEnvelopeForTest(t, stdout)
		var data internalclob.MarketTradesProbeResult
		if err := json.Unmarshal(got.Data, &data); err != nil {
			t.Fatalf("data is not probe payload: %v\n%s", err, got.Data)
		}
		if data.Classification != internalclob.ProbeMarketWide || data.RowCount != 2 {
			t.Fatalf("data=%+v", data)
		}
	})
}

func executeCLOBAuthHelperForTest(args []string, balances *fakeCLOBBalanceRunner, accounts *fakeCLOBAccountReadRunner, diagnostics *fakeCLOBDiagnosticRunner) (string, string, error) {
	clobCmd := commandGroup("clob", "CLOB authenticated reads")
	addCLOBAuthenticatedReadCommands(clobCmd, balances, accounts, diagnostics)

	root := &cobra.Command{Use: "polygolem", SilenceUsage: true, SilenceErrors: true}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.PersistentFlags().Bool("json", false, "emit JSON output")
	root.AddCommand(clobCmd)
	root.SetArgs(args)
	installJSONContract(root)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

func TestCLOBReadOnlyMarketDataCommandsKeepFlags(t *testing.T) {
	root := NewRootCommand(Options{Version: "test", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	for _, tc := range []struct {
		args  []string
		flags []string
	}{
		{args: []string{"clob", "book"}, flags: []string{"output"}},
		{args: []string{"clob", "tick-size"}, flags: []string{"output"}},
		{args: []string{"clob", "price-history"}, flags: []string{"output", "interval"}},
		{args: []string{"clob", "market"}, flags: []string{"output"}},
		{args: []string{"clob", "market-by-token"}, flags: []string{"output"}},
		{args: []string{"clob", "markets"}, flags: []string{"output", "cursor"}},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			cmd, _, err := root.Find(tc.args)
			if err != nil {
				t.Fatalf("Find returned error: %v", err)
			}
			for _, flag := range tc.flags {
				if cmd.Flags().Lookup(flag) == nil {
					t.Fatalf("%s missing --%s flag", strings.Join(tc.args, " "), flag)
				}
			}
		})
	}
}

func TestCLOBReadOnlyMarketDataValidatesOutputBeforeNetwork(t *testing.T) {
	root := NewRootCommand(Options{Version: "test", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	root.SetArgs([]string{"clob", "book", "token-1", "--output", "table"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute returned nil error")
	}
	if !strings.Contains(err.Error(), "only --output json is supported") {
		t.Fatalf("error=%q, want output validation error", err.Error())
	}
}

func TestCLOBAuthenticatedReadCommandsKeepFlags(t *testing.T) {
	root := NewRootCommand(Options{Version: "test", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	for _, tc := range []struct {
		args  []string
		flags []string
	}{
		{args: []string{"clob", "balance"}, flags: []string{"output", "asset-type", "token-id"}},
		{args: []string{"clob", "update-balance"}, flags: []string{"output", "asset-type", "token-id"}},
		{args: []string{"clob", "list-builder-fee-keys"}, flags: []string{"output"}},
		{args: []string{"clob", "market-trades-probe"}, flags: []string{"output", "market", "asset-id", "cursor"}},
		{args: []string{"clob", "orders"}, flags: []string{"output"}},
		{args: []string{"clob", "order"}, flags: []string{"output"}},
		{args: []string{"clob", "trades"}, flags: []string{"output"}},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			cmd, _, err := root.Find(tc.args)
			if err != nil {
				t.Fatalf("Find returned error: %v", err)
			}
			for _, flag := range tc.flags {
				if cmd.Flags().Lookup(flag) == nil {
					t.Fatalf("%s missing --%s flag", strings.Join(tc.args, " "), flag)
				}
			}
		})
	}
}

func TestCLOBAuthenticatedReadValidatesOutputBeforeLoadingPrivateKey(t *testing.T) {
	t.Setenv("POLYMARKET_PRIVATE_KEY", "")
	for _, args := range [][]string{
		{"clob", "balance", "--output", "table"},
		{"clob", "list-builder-fee-keys", "--output", "table"},
		{"clob", "market-trades-probe", "--output", "table"},
		{"clob", "orders", "--output", "table"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			root := NewRootCommand(Options{Version: "test", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
			root.SetArgs(args)

			err := root.Execute()
			if err == nil {
				t.Fatal("Execute returned nil error")
			}
			if !strings.Contains(err.Error(), "only --output json is supported") {
				t.Fatalf("error=%q, want output validation error", err.Error())
			}
			if strings.Contains(err.Error(), "POLYMARKET_PRIVATE_KEY") {
				t.Fatalf("private key was loaded before output validation: %q", err.Error())
			}
		})
	}
}
