package cli

import (
	"bytes"
	"testing"
)

func TestCommandRenameContract(t *testing.T) {
	root := NewRootCommand(Options{Version: "test", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	for _, path := range [][]string{
		{"ping"},
		{"markets", "search"},
		{"book", "get"},
		{"exchange", "simulate"},
		{"analytics", "positions"},
		{"wallet", "derive"},
		{"sim", "trade"},
		{"prices", "live"},
		{"credentials", "status"},
		{"risk", "status"},
		{"doctor"},
		{"debug"},
		{"check-upstream", "llms"},
		{"tx", "transaction"},
		{"builder-keys", "auto"},
	} {
		if cmd, _, err := root.Find(path); err != nil || cmd == nil {
			t.Fatalf("new command %v not registered: cmd=%v err=%v", path, cmd, err)
		}
	}
}

func TestLegacyCommandNamesAreRemoved(t *testing.T) {
	root := NewRootCommand(Options{Version: "test", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	for _, path := range [][]string{
		{"health"},
		{"discover"},
		{"orderbook"},
		{"clob"},
		{"clob", "simulate-order"},
		{"data"},
		{"intel"},
		{"deposit-wallet"},
		{"auth"},
		{"builder"},
		{"paper"},
		{"marketdata"},
		{"live"},
		{"preflight"},
		{"diag"},
		{"drift"},
		{"relayer"},
	} {
		if cmd, _, err := root.Find(path); err == nil && cmd != root {
			t.Fatalf("legacy command %v is still registered as %q", path, cmd.CommandPath())
		}
	}
}
