package cli

import "testing"

func TestRelayerTransactionCommandIsRegistered(t *testing.T) {
	root := NewRootCommand(Options{Version: "test-version"})
	cmd, _, err := root.Find([]string{"relayer", "transaction", "tx-1"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("relayer transaction command missing")
	}
	if cmd.Use != "transaction <tx-id>" {
		t.Fatalf("Use=%q", cmd.Use)
	}
}
