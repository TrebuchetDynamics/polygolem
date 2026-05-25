package cli

import "testing"

func TestDiscoverCryptoCommandKeepsSearchFlags(t *testing.T) {
	root := NewRootCommand(Options{Version: "test-version"})
	cmd, _, err := root.Find([]string{"discover", "crypto"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("discover crypto command missing")
	}
	for _, name := range []string{"asset", "interval", "limit", "enrich"} {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("%s flag missing", name)
		}
	}
}

func TestDiscoverCryptoWindowCommandKeepsFlags(t *testing.T) {
	root := NewRootCommand(Options{Version: "test-version"})
	cmd, _, err := root.Find([]string{"discover", "crypto-window"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("discover crypto-window command missing")
	}
	for _, name := range []string{"asset", "interval", "enrich"} {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("%s flag missing", name)
		}
	}
}

func TestDiscoverCrypto5mCommandKeepsEnrichFlag(t *testing.T) {
	root := NewRootCommand(Options{Version: "test-version"})
	cmd, _, err := root.Find([]string{"discover", "crypto-5m"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("discover crypto-5m command missing")
	}
	if flag := cmd.Flags().Lookup("enrich"); flag == nil {
		t.Fatal("enrich flag missing")
	}
}
