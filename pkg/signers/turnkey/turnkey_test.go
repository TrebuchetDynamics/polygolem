package turnkeysigner

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/signers"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type fakeTurnkeyBackend struct {
	organizationID string
	walletID       string
	address        string
	ctxDeadline    bool
}

func (f *fakeTurnkeyBackend) SignHash(ctx context.Context, organizationID, walletID, address string, hash [32]byte) ([]byte, error) {
	f.capture(ctx, organizationID, walletID, address)
	return []byte(strings.Repeat("c", 65)), nil
}

func (f *fakeTurnkeyBackend) SignTypedData(ctx context.Context, organizationID, walletID, address string, domainHash, structHash [32]byte) ([32]byte, error) {
	f.capture(ctx, organizationID, walletID, address)
	var out [32]byte
	out[0] = 0x77
	return out, nil
}

func (f *fakeTurnkeyBackend) SignEIP712(ctx context.Context, organizationID, walletID, address string, typed apitypes.TypedData) ([]byte, error) {
	f.capture(ctx, organizationID, walletID, address)
	return []byte(strings.Repeat("d", 65)), nil
}

func (f *fakeTurnkeyBackend) capture(ctx context.Context, organizationID, walletID, address string) {
	f.organizationID = organizationID
	f.walletID = walletID
	f.address = address
	_, f.ctxDeadline = ctx.Deadline()
}

func TestTurnkeySignerDelegatesWithIdentityAndTimeout(t *testing.T) {
	backend := &fakeTurnkeyBackend{}
	signer, err := New(Config{OrganizationID: "org-1", WalletID: "wallet-1", Address: "0xabc", ChainID: 137, Timeout: time.Second, Backend: backend})
	if err != nil {
		t.Fatal(err)
	}
	var _ signers.Signer = signer
	var hash [32]byte
	sig, err := signer.SignHash(hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 65 {
		t.Fatalf("signature length=%d", len(sig))
	}
	if backend.organizationID != "org-1" || backend.walletID != "wallet-1" || backend.address != "0xabc" || !backend.ctxDeadline {
		t.Fatalf("bad delegation: %+v", backend)
	}
	got, err := signer.SignTypedData(hash, hash)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 0x77 {
		t.Fatalf("typed result=%x", got)
	}
}

func TestTurnkeySignerRejectsMissingConfig(t *testing.T) {
	backend := &fakeTurnkeyBackend{}
	cases := []Config{
		{WalletID: "wallet", Address: "0xabc", Backend: backend},
		{OrganizationID: "org", Address: "0xabc", Backend: backend},
		{OrganizationID: "org", WalletID: "wallet", Backend: backend},
		{OrganizationID: "org", WalletID: "wallet", Address: "0xabc"},
	}
	for _, cfg := range cases {
		if _, err := New(cfg); err == nil {
			t.Fatalf("expected validation error for %+v", cfg)
		}
	}
}
