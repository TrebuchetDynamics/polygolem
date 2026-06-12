package cli

import "testing"

func TestStreamMarketCommandKeepsStreamFlags(t *testing.T) {
	root := NewRootCommand(Options{Version: "test-version"})
	cmd, _, err := root.Find([]string{"stream", "market"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("stream market command missing")
	}
	for _, name := range []string{"asset-ids", "url", "max-messages", "custom-features", "level", "stats"} {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("%s flag missing", name)
		}
	}
}

func TestStreamUserCommandKeepsAuthStreamFlags(t *testing.T) {
	root := NewRootCommand(Options{Version: "test-version"})
	cmd, _, err := root.Find([]string{"stream", "user"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("stream user command missing")
	}
	for _, name := range []string{"markets", "url", "max-messages", "stats"} {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("%s flag missing", name)
		}
	}
}

func TestStreamCryptoCommandKeepsDiscoveryAndStreamFlags(t *testing.T) {
	root := NewRootCommand(Options{Version: "test-version"})
	cmd, _, err := root.Find([]string{"stream", "crypto"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("stream crypto command missing")
	}
	for _, name := range []string{"asset", "interval", "max-messages", "custom-features", "stats"} {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("%s flag missing", name)
		}
	}
}
