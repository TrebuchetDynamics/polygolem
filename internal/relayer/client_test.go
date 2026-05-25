package relayer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
)

func TestRelayerTransactionState_IsTerminal(t *testing.T) {
	tests := []struct {
		state    RelayerTransactionState
		terminal bool
		success  bool
	}{
		{StateNew, false, false},
		{StateExecuted, false, false},
		{StateMined, true, true},
		{StateConfirmed, true, true},
		{StateFailed, true, false},
		{StateInvalid, true, false},
	}
	for _, tt := range tests {
		if got := tt.state.IsTerminal(); got != tt.terminal {
			t.Errorf("%s.IsTerminal() = %v, want %v", tt.state, got, tt.terminal)
		}
		if got := tt.state.IsSuccess(); got != tt.success {
			t.Errorf("%s.IsSuccess() = %v, want %v", tt.state, got, tt.success)
		}
	}
}

func TestDeployedResponseDecodesStringBooleanAndNumericAddress(t *testing.T) {
	var resp DeployedResponse
	if err := json.Unmarshal([]byte(`{"deployed":"true","address":123}`), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Deployed || resp.Address != "123" {
		t.Fatalf("unexpected deployed response: %+v", resp)
	}
}

func TestNonceResponseDecodesNumericNonce(t *testing.T) {
	var resp NonceResponse
	if err := json.Unmarshal([]byte(`{"nonce":7}`), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Nonce != "7" {
		t.Fatalf("Nonce=%q", resp.Nonce)
	}
}

func TestRelayerTransactionDecodesSnakeCaseAliases(t *testing.T) {
	var tx RelayerTransaction
	raw := `{"transaction_id":"tx-1","transaction_hash":"0xabc","proxy_address":"0xwallet","nonce":7,"value":0,"state":"STATE_MINED","type":"WALLET","created_at":"2026-05-24T00:00:00Z","updated_at":"2026-05-24T00:01:00Z"}`
	if err := json.Unmarshal([]byte(raw), &tx); err != nil {
		t.Fatal(err)
	}
	if tx.TransactionID != "tx-1" || tx.TransactionHash != "0xabc" || tx.ProxyAddress != "0xwallet" {
		t.Fatalf("unexpected decoded transaction: %+v", tx)
	}
	if tx.Nonce != "7" || tx.Value != "0" || tx.CreatedAt == "" || tx.UpdatedAt == "" {
		t.Fatalf("unexpected scalar fields: %+v", tx)
	}
}

func TestRelayerTransactionDecodesLowerCamelIDAlias(t *testing.T) {
	var tx RelayerTransaction
	raw := `{"transactionId":"tx-lower-camel","transactionHash":"0xabc","state":"STATE_MINED","type":"WALLET"}`
	if err := json.Unmarshal([]byte(raw), &tx); err != nil {
		t.Fatal(err)
	}
	if tx.TransactionID != "tx-lower-camel" || tx.TransactionHash != "0xabc" {
		t.Fatalf("unexpected decoded transaction: %+v", tx)
	}
}

func TestNew_RequiresBuilderConfig(t *testing.T) {
	_, err := New("https://relayer.example.com", auth.BuilderConfig{}, 137)
	if err == nil {
		t.Fatal("expected error for empty builder config")
	}
}

func TestNew_DefaultChainID(t *testing.T) {
	bc := auth.BuilderConfig{Key: "k", Secret: "s", Passphrase: "p"}
	c, err := New("https://relayer.example.com", bc, 0)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if c.chainID != 137 {
		t.Errorf("expected default chainID 137, got %d", c.chainID)
	}
}

func TestClient_RequiresValidAddress(t *testing.T) {
	bc := auth.BuilderConfig{Key: "k", Secret: "s", Passphrase: "p"}
	c, err := New("https://relayer.example.com", bc, 137)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if _, err := c.SubmitWalletCreate(nil, ""); err == nil {
		t.Fatal("expected error for empty owner address")
	}
	if _, err := c.GetNonce(nil, ""); err == nil {
		t.Fatal("expected error for empty nonce address")
	}
	if _, err := c.GetTransaction(nil, ""); err == nil {
		t.Fatal("expected error for empty tx ID")
	}
	if _, err := c.IsDeployed(nil, ""); err == nil {
		t.Fatal("expected error for empty deployed address")
	}
	if _, err := c.SubmitWalletBatch(nil, "", "", "", "", "", nil); err == nil {
		t.Fatal("expected error for empty wallet batch params")
	}
	if _, err := c.SubmitWalletBatch(nil, "0x1", "0x2", "1", "0xsig", "99999", []DepositWalletCall{}); err == nil {
		t.Fatal("expected error for empty calls")
	}
}

func TestClient_GetTransactionAcceptsArrayResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method=%s, want GET", r.Method)
		}
		if r.URL.Path != "/transaction" {
			t.Errorf("path=%q, want /transaction", r.URL.Path)
		}
		if got := r.URL.Query().Get("id"); got != "tx-1" {
			t.Errorf("id=%q, want tx-1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"transactionID":"tx-1","transactionHash":"0xabc","state":"STATE_CONFIRMED","type":"WALLET-CREATE"}]`))
	}))
	defer srv.Close()

	bc := auth.BuilderConfig{Key: "k", Secret: "s", Passphrase: "p"}
	c, err := New(srv.URL, bc, 137)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	tx, err := c.GetTransaction(context.Background(), "tx-1")
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}
	if tx.TransactionID != "tx-1" || tx.TransactionHash != "0xabc" || tx.State != "STATE_CONFIRMED" {
		t.Fatalf("unexpected transaction: %+v", tx)
	}
}

func TestClient_GetTransactionRejectsEmptyArrayResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	bc := auth.BuilderConfig{Key: "k", Secret: "s", Passphrase: "p"}
	c, err := New(srv.URL, bc, 137)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	_, err = c.GetTransaction(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error=%q, want transaction ID", err.Error())
	}
}
