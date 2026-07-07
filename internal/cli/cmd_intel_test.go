package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	intelworkflow "github.com/TrebuchetDynamics/polygolem/internal/intel"
	sdkintel "github.com/TrebuchetDynamics/polygolem/pkg/intel"
	"github.com/spf13/cobra"
)

type fakeIntelRunner struct {
	wallet        string
	dossierOpts   intelworkflow.DossierOptions
	leaderOpts    intelworkflow.LeaderboardOptions
	dossierResult *sdkintel.WalletDossier
	leaderResult  []sdkintel.LeaderboardRow
	alertsOpts    intelworkflow.AlertOptions
	alertsResult  []sdkintel.Signal
	flowMarket    string
	flowOpts      intelworkflow.MarketFlowOptions
	flowResult    *sdkintel.MarketFlow
}

func (f *fakeIntelRunner) WalletDossier(_ context.Context, wallet string, opts intelworkflow.DossierOptions) (*sdkintel.WalletDossier, error) {
	f.wallet = wallet
	f.dossierOpts = opts
	return f.dossierResult, nil
}

func (f *fakeIntelRunner) Leaderboard(_ context.Context, opts intelworkflow.LeaderboardOptions) ([]sdkintel.LeaderboardRow, error) {
	f.leaderOpts = opts
	return f.leaderResult, nil
}

func (f *fakeIntelRunner) Alerts(_ context.Context, opts intelworkflow.AlertOptions) ([]sdkintel.Signal, error) {
	f.alertsOpts = opts
	return f.alertsResult, nil
}

func (f *fakeIntelRunner) MarketFlow(_ context.Context, market string, opts intelworkflow.MarketFlowOptions) (*sdkintel.MarketFlow, error) {
	f.flowMarket = market
	f.flowOpts = opts
	return f.flowResult, nil
}

func TestIntelWalletCommandDelegatesToRunnerAndJSONEnvelope(t *testing.T) {
	fake := &fakeIntelRunner{dossierResult: &sdkintel.WalletDossier{Wallet: "0xwallet", Status: sdkintel.DossierStatusComplete}}
	root := intelTestRoot(fake)
	root.SetArgs([]string{"--json", "wallets", "wallet", "0xwallet", "--limit", "7"})
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if fake.wallet != "0xwallet" || fake.dossierOpts.Limit != 7 {
		t.Fatalf("request wallet=%q opts=%+v", fake.wallet, fake.dossierOpts)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "wallets wallet" {
		t.Fatalf("meta.command=%q", got.Meta.Command)
	}
	var dossier sdkintel.WalletDossier
	if err := json.Unmarshal(got.Data, &dossier); err != nil {
		t.Fatalf("data is not dossier: %v\n%s", err, got.Data)
	}
	if dossier.Wallet != "0xwallet" || dossier.Status != sdkintel.DossierStatusComplete {
		t.Fatalf("dossier=%+v", dossier)
	}
}

func TestIntelLeaderboardCommandDelegatesToRunner(t *testing.T) {
	fake := &fakeIntelRunner{leaderResult: []sdkintel.LeaderboardRow{{Rank: 1, Wallet: "0xwallet"}}}
	root := intelTestRoot(fake)
	root.SetArgs([]string{"--json", "wallets", "leaderboard", "--limit", "5"})
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if fake.leaderOpts.Limit != 5 {
		t.Fatalf("opts=%+v", fake.leaderOpts)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "wallets leaderboard" {
		t.Fatalf("meta.command=%q", got.Meta.Command)
	}
	var rows []sdkintel.LeaderboardRow
	if err := json.Unmarshal(got.Data, &rows); err != nil {
		t.Fatalf("data is not leaderboard rows: %v\n%s", err, got.Data)
	}
	if len(rows) != 1 || rows[0].Wallet != "0xwallet" {
		t.Fatalf("rows=%+v", rows)
	}
}

func TestIntelAlertsCommandDelegatesToRunner(t *testing.T) {
	fake := &fakeIntelRunner{alertsResult: []sdkintel.Signal{{Wallet: "0xwallet", Score: 80}}}
	root := intelTestRoot(fake)
	root.SetArgs([]string{"--json", "wallets", "alerts", "--user", "0xwallet", "--limit", "9", "--min-score", "75"})
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if fake.alertsOpts.User != "0xwallet" || fake.alertsOpts.Limit != 9 || fake.alertsOpts.MinScore != 75 {
		t.Fatalf("opts=%+v", fake.alertsOpts)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "wallets alerts" {
		t.Fatalf("meta.command=%q", got.Meta.Command)
	}
	var payload sdkintel.DossierAlerts
	if err := json.Unmarshal(got.Data, &payload); err != nil {
		t.Fatalf("data is not dossier alerts payload: %v\n%s", err, got.Data)
	}
	if len(payload.DossierAlerts) != 1 || payload.DossierAlerts[0].Score != 80 {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestIntelMarketFlowCommandDelegatesToRunner(t *testing.T) {
	fake := &fakeIntelRunner{flowResult: &sdkintel.MarketFlow{Market: "0xmarket", TradeCount: 2}}
	root := intelTestRoot(fake)
	root.SetArgs([]string{"--json", "wallets", "market-flow", "0xmarket", "--limit", "11"})
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if fake.flowMarket != "0xmarket" || fake.flowOpts.Limit != 11 {
		t.Fatalf("market=%q opts=%+v", fake.flowMarket, fake.flowOpts)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "wallets market-flow" {
		t.Fatalf("meta.command=%q", got.Meta.Command)
	}
	var flow sdkintel.MarketFlow
	if err := json.Unmarshal(got.Data, &flow); err != nil {
		t.Fatalf("data is not market flow: %v\n%s", err, got.Data)
	}
	if flow.Market != "0xmarket" || flow.TradeCount != 2 {
		t.Fatalf("flow=%+v", flow)
	}
}

func TestIntelLeaderboardRejectsUnsupportedSort(t *testing.T) {
	root := intelTestRoot(&fakeIntelRunner{})
	root.SetArgs([]string{"wallets", "leaderboard", "--sort", "shrinkage-win-rate"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute returned nil error")
	}
	if !strings.Contains(err.Error(), "unsupported --sort") {
		t.Fatalf("error=%q", err.Error())
	}
}

func TestRootHasIntelCommands(t *testing.T) {
	root := NewRootCommand(Options{Version: "test", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	for _, args := range [][]string{{"wallets", "wallet"}, {"wallets", "leaderboard"}, {"wallets", "alerts"}, {"wallets", "market-flow"}} {
		cmd, _, err := root.Find(args)
		if err != nil {
			t.Fatalf("Find(%v) returned error: %v", args, err)
		}
		if cmd == nil {
			t.Fatalf("Find(%v) returned nil", args)
		}
	}
}

func intelTestRoot(fake *fakeIntelRunner) *cobra.Command {
	root := &cobra.Command{Use: "polygolem", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().Bool("json", false, "emit JSON output")
	root.AddCommand(newIntelCommand(fake))
	installJSONContract(root)
	return root
}
