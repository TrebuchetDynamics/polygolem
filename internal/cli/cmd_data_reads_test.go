package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/dataapi"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/datareads"
	"github.com/spf13/cobra"
)

type fakeDataReadsRunner struct {
	request datareads.Request
	result  any
}

func (f *fakeDataReadsRunner) Run(_ context.Context, req datareads.Request) (any, error) {
	f.request = req
	return f.result, nil
}

func TestDataTradesCommandDelegatesToRunnerAndJSONEnvelope(t *testing.T) {
	fake := &fakeDataReadsRunner{result: []dataapi.Trade{{ID: "trade-1", AssetID: "asset-1"}}}
	root := &cobra.Command{Use: "polygolem", SilenceUsage: true, SilenceErrors: true}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.PersistentFlags().Bool("json", false, "emit JSON output")
	root.AddCommand(newDataCommand(fake))
	root.SetArgs([]string{"--json", "data", "trades", "--user", "0xuser", "--limit", "7"})
	installJSONContract(root)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr.String())
	}
	if fake.request.Operation != datareads.Trades || fake.request.User != "0xuser" || fake.request.Limit != 7 {
		t.Fatalf("request=%+v", fake.request)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "data trades" {
		t.Fatalf("meta.command=%q, want data trades", got.Meta.Command)
	}
	var data []dataapi.Trade
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("data is not trades payload: %v\n%s", err, got.Data)
	}
	if len(data) != 1 || data[0].ID != "trade-1" || data[0].AssetID != "asset-1" {
		t.Fatalf("data=%+v", data)
	}
}
