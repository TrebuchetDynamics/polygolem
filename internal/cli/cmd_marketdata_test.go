package cli

import "testing"

func TestMarketDataLiveCommandKeepsStreamFlags(t *testing.T) {
	root := NewRootCommand(Options{Version: "test-version"})
	cmd, _, err := root.Find([]string{"marketdata", "live"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("marketdata live command missing")
	}
	for _, name := range []string{"asset-ids", "url", "max-messages", "custom-features", "level"} {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("%s flag missing", name)
		}
	}
}

func TestMarketDataCryptoCommandKeepsSnapshotFlags(t *testing.T) {
	root := NewRootCommand(Options{Version: "test-version"})
	cmd, _, err := root.Find([]string{"marketdata", "crypto"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("marketdata crypto command missing")
	}
	for _, name := range []string{"asset", "interval", "limit"} {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("%s flag missing", name)
		}
	}
}
