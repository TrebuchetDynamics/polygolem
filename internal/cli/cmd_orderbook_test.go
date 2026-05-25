package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/workflows/orderbookreads"
	"github.com/spf13/cobra"
)

type fakeOrderbookRunner struct {
	request orderbookreads.Request
	result  any
}

func (f *fakeOrderbookRunner) Run(_ context.Context, req orderbookreads.Request) (any, error) {
	f.request = req
	return f.result, nil
}

func TestOrderbookCommandDelegatesToRunnerAndJSONEnvelope(t *testing.T) {
	fake := &fakeOrderbookRunner{result: map[string]string{"token_id": "token-1", "price": "0.42"}}
	root := &cobra.Command{Use: "polygolem", SilenceUsage: true, SilenceErrors: true}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.PersistentFlags().Bool("json", false, "emit JSON output")
	root.AddCommand(newOrderbookCommand(fake))
	root.SetArgs([]string{"--json", "orderbook", "price", "--token-id", "token-1"})
	installJSONContract(root)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr.String())
	}
	if fake.request.Operation != orderbookreads.Price || fake.request.TokenID != "token-1" {
		t.Fatalf("request=%+v", fake.request)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "orderbook price" {
		t.Fatalf("meta.command=%q, want orderbook price", got.Meta.Command)
	}
	var data map[string]string
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("data is not orderbook payload: %v\n%s", err, got.Data)
	}
	if !reflect.DeepEqual(data, map[string]string{"token_id": "token-1", "price": "0.42"}) {
		t.Fatalf("data=%+v", data)
	}
}
