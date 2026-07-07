package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/discoverreads"
	"github.com/spf13/cobra"
)

type fakeDiscoverReadsRunner struct {
	request discoverreads.Request
	result  any
}

func (f *fakeDiscoverReadsRunner) Run(_ context.Context, req discoverreads.Request) (any, error) {
	f.request = req
	return f.result, nil
}

func TestDiscoverSearchCommandDelegatesToRunnerAndJSONEnvelope(t *testing.T) {
	fake := &fakeDiscoverReadsRunner{result: &polytypes.SearchResponse{Events: []polytypes.Event{{ID: "event-1", Slug: "event-one"}}}}
	root := &cobra.Command{Use: "polygolem", SilenceUsage: true, SilenceErrors: true}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.PersistentFlags().Bool("json", false, "emit JSON output")
	root.AddCommand(newDiscoverCommand(nil, fake))
	root.SetArgs([]string{"--json", "markets", "search", "--query", "btc", "--limit", "5"})
	installJSONContract(root)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr.String())
	}
	if fake.request.Operation != discoverreads.Search || fake.request.Query != "btc" || fake.request.Limit != 5 {
		t.Fatalf("request=%+v", fake.request)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "markets search" {
		t.Fatalf("meta.command=%q, want markets search", got.Meta.Command)
	}
	var data polytypes.SearchResponse
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("data is not search payload: %v\n%s", err, got.Data)
	}
	if len(data.Events) != 1 || data.Events[0].Slug != "event-one" {
		t.Fatalf("data=%+v", data)
	}
}
