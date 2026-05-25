package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

type fakePaperPricer struct {
	tokenID string
	side    string
}

func (f *fakePaperPricer) Price(_ context.Context, tokenID, side string) (string, error) {
	f.tokenID = tokenID
	f.side = side
	return "0.42", nil
}

func TestPaperBuyCommandDelegatesPricingAndJSONEnvelope(t *testing.T) {
	pricer := &fakePaperPricer{}
	root := &cobra.Command{Use: "polygolem", SilenceUsage: true, SilenceErrors: true}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.PersistentFlags().Bool("json", false, "emit JSON output")
	root.AddCommand(newPaperCommand(true, pricer))
	root.SetArgs([]string{"--json", "paper", "buy", "--token-id", "token-1", "--size", "2"})
	installJSONContract(root)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr.String())
	}
	if pricer.tokenID != "token-1" || pricer.side != "SELL" {
		t.Fatalf("price call token=%q side=%q", pricer.tokenID, pricer.side)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "paper buy" {
		t.Fatalf("meta.command=%q, want paper buy", got.Meta.Command)
	}
	var data struct {
		Action string  `json:"action"`
		Price  float64 `json:"price"`
		Size   float64 `json:"size"`
		Cost   float64 `json:"cost"`
		Cash   float64 `json:"cash"`
	}
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("data is not paper buy payload: %v\n%s", err, got.Data)
	}
	if data.Action != "buy" || data.Price != 0.42 || data.Size != 2 || data.Cost != 0.84 || data.Cash != 9999.16 {
		t.Fatalf("data=%+v", data)
	}
}
