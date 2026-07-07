package cli

import (
	"encoding/json"
	"testing"
)

func TestPaperTradeTokenBypassUsesJSONEnvelope(t *testing.T) {
	stdout, stderr, err := executeRootForTest("--json", "sim", "trade", "--token-id", "token-1", "--price", "0.25", "--size", "4")
	if err != nil {
		t.Fatalf("Execute returned error: %v\nstderr:\n%s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr=%q, want empty", stderr)
	}
	got := parseJSONEnvelopeForTest(t, stdout)
	if !got.OK {
		t.Fatalf("ok=false, want true\nenvelope=%s", stdout)
	}
	if got.Meta.Command != "sim trade" {
		t.Fatalf("meta.command=%q, want sim trade", got.Meta.Command)
	}

	var data struct {
		Action  string  `json:"action"`
		TokenID string  `json:"token_id"`
		Price   float64 `json:"price"`
		Size    float64 `json:"size"`
		Cost    float64 `json:"cost"`
		Cash    float64 `json:"cash"`
	}
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("data is not paper-trade payload: %v\n%s", err, got.Data)
	}
	if data.Action != "paper_trade" || data.TokenID != "token-1" {
		t.Fatalf("unexpected action/token: %+v", data)
	}
	if data.Price != 0.25 || data.Size != 4 || data.Cost != 1 || data.Cash != 9999 {
		t.Fatalf("unexpected sim trade accounting: %+v", data)
	}
}

func TestPaperCryptoCommandKeepsDiscoveryFlags(t *testing.T) {
	root := NewRootCommand(Options{Version: "test-version"})
	cmd, _, err := root.Find([]string{"sim", "crypto"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("paper crypto command missing")
	}
	for _, name := range []string{"asset", "interval", "limit"} {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("%s flag missing", name)
		}
	}
}
