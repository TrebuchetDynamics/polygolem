package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/workflows/healthcheck"
)

type fakeHealthRunner struct {
	result healthcheck.Result
	called bool
}

func (f *fakeHealthRunner) Run(context.Context) healthcheck.Result {
	f.called = true
	return f.result
}

func TestHealthCommandUsesRunnerAndJSONEnvelope(t *testing.T) {
	fake := &fakeHealthRunner{result: healthcheck.Result{"gamma": "ok", "clob": "clob down"}}
	cmd := newHealthCommand(fake)
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.PersistentFlags().Bool("json", false, "emit JSON output")
	cmd.SetArgs([]string{"--json"})
	installJSONContract(cmd)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr.String())
	}
	if !fake.called {
		t.Fatal("health runner was not called")
	}
	got := parseJSONEnvelopeForTest(t, stdout.String())
	if got.Meta.Command != "health" {
		t.Fatalf("meta.command=%q, want health", got.Meta.Command)
	}
	var data map[string]string
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("data is not health payload: %v\n%s", err, got.Data)
	}
	if !reflect.DeepEqual(data, map[string]string{"gamma": "ok", "clob": "clob down"}) {
		t.Fatalf("data=%+v", data)
	}
}
