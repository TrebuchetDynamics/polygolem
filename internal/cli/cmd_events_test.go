package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/eventreads"
	"github.com/spf13/cobra"
)

type fakeEventsRunner struct {
	request eventreads.Request
	result  []polytypes.Event
}

func (f *fakeEventsRunner) Run(_ context.Context, req eventreads.Request) ([]polytypes.Event, error) {
	f.request = req
	return f.result, nil
}

func TestEventsListCommandDelegatesToRunnerAndJSONEnvelope(t *testing.T) {
	fake := &fakeEventsRunner{result: []polytypes.Event{{ID: "evt-1", Slug: "event-one", Title: "Event One"}}}
	root := &cobra.Command{Use: "polygolem", SilenceUsage: true, SilenceErrors: true}
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.PersistentFlags().Bool("json", false, "emit JSON output")
	root.AddCommand(newEventsCommand(fake))
	root.SetArgs([]string{"--json", "events", "list"})
	installJSONContract(root)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr.String())
	}
	if fake.request.Limit != 0 {
		t.Fatalf("limit=%d, want default 0", fake.request.Limit)
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "events list" {
		t.Fatalf("meta.command=%q, want events list", got.Meta.Command)
	}
	var data []polytypes.Event
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("data is not events payload: %v\n%s", err, got.Data)
	}
	if len(data) != 1 || data[0].ID != "evt-1" || data[0].Slug != "event-one" {
		t.Fatalf("data=%+v", data)
	}
}
