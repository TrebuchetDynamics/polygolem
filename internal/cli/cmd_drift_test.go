package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const completeLLMSText = `
https://docs.polymarket.com/market-data/overview
https://docs.polymarket.com/trading/overview
https://docs.polymarket.com/api-reference/core/get-current-positions-for-a-user
https://docs.polymarket.com/api-reference/relayer/submit-a-transaction
https://docs.polymarket.com/trading/bridge/deposit
https://docs.polymarket.com/market-data/websocket/overview
`

func TestDriftLLMSCommandReadsSavedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "llms.txt")
	if err := os.WriteFile(path, []byte(completeLLMSText), 0o600); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	root := NewRootCommand(Options{Version: "test", Stdout: out, Stderr: &bytes.Buffer{}})
	root.SetArgs([]string{"check-upstream", "llms", "--file", path})
	if err := root.Execute(); err != nil {
		t.Fatalf("drift llms failed: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "upstream drift: ok") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestDriftLLMSCommandFailsOnMissingSurface(t *testing.T) {
	path := filepath.Join(t.TempDir(), "llms.txt")
	if err := os.WriteFile(path, []byte("https://docs.polymarket.com/trading/overview"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	root := NewRootCommand(Options{Version: "test", Stdout: out, Stderr: &bytes.Buffer{}})
	root.SetArgs([]string{"check-upstream", "llms", "--file", path})
	err := root.Execute()
	if err == nil {
		t.Fatalf("expected drift failure, output=%s", out.String())
	}
	if !strings.Contains(err.Error(), "missing official Polymarket docs") || !strings.Contains(out.String(), "gamma.markets") {
		t.Fatalf("missing-surface output not useful: err=%v out=%s", err, out.String())
	}
}

func TestDriftLLMSCommandSupportsJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "llms.txt")
	if err := os.WriteFile(path, []byte(completeLLMSText), 0o600); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	root := NewRootCommand(Options{Version: "test", Stdout: out, Stderr: &bytes.Buffer{}})
	root.SetArgs([]string{"--json", "check-upstream", "llms", "--file", path})
	if err := root.Execute(); err != nil {
		t.Fatalf("drift llms json failed: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"ok": true`) || !strings.Contains(out.String(), `"checked"`) {
		t.Fatalf("unexpected json: %s", out.String())
	}
}
