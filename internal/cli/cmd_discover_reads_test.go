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

func TestDiscoverCategoriesCommandDelegatesToRunnerAndJSONEnvelope(t *testing.T) {
	fake := &fakeDiscoverReadsRunner{result: &polytypes.CategoryEventsResponse{Category: polytypes.PolymarketCategory{Slug: "mentions"}, Events: []polytypes.Event{{ID: "event-1", Slug: "event-one"}}, NextCursor: "cursor-2", HasMore: true}}
	root := &cobra.Command{Use: "polygolem", SilenceUsage: true, SilenceErrors: true}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.PersistentFlags().Bool("json", false, "emit JSON output")
	root.AddCommand(newDiscoverCommand(nil, fake))
	root.SetArgs([]string{"--json", "markets", "categories", "--slug", "mentions", "--events", "--limit", "5", "--cursor", "cursor-1"})
	installJSONContract(root)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr.String())
	}
	if fake.request.Operation != discoverreads.CategoryEvents || fake.request.Slug != "mentions" || fake.request.Limit != 5 || fake.request.Cursor != "cursor-1" {
		t.Fatalf("request=%+v", fake.request)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "markets categories" {
		t.Fatalf("meta.command=%q, want markets categories", got.Meta.Command)
	}
	var data polytypes.CategoryEventsResponse
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("data is not category events payload: %v\n%s", err, got.Data)
	}
	if data.Category.Slug != "mentions" || len(data.Events) != 1 || data.NextCursor != "cursor-2" {
		t.Fatalf("data=%+v", data)
	}
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
