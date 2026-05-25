package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDataOrderResultsCommandKeepsAuditFlags(t *testing.T) {
	root := NewRootCommand(Options{Version: "test", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	cmd, _, err := root.Find([]string{"data", "order-results"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	for _, flag := range []string{"user", "limit", "include-clob"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("data order-results missing --%s flag", flag)
		}
	}
}

func TestDataOrderResultsValidatesUserBeforeLoadingPrivateKey(t *testing.T) {
	t.Setenv("POLYMARKET_PRIVATE_KEY", "")
	root := NewRootCommand(Options{Version: "test", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	root.SetArgs([]string{"data", "order-results", "--include-clob"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute returned nil error")
	}
	if !strings.Contains(err.Error(), "--user required") {
		t.Fatalf("error=%q, want --user required", err.Error())
	}
	if strings.Contains(err.Error(), "POLYMARKET_PRIVATE_KEY") {
		t.Fatalf("private key was loaded before user validation: %q", err.Error())
	}
}
