package kmssigner

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/signers"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type fakeBackend struct {
	keyID       string
	ctxDeadline bool
}

func (f *fakeBackend) SignHash(ctx context.Context, keyID string, hash [32]byte) ([]byte, error) {
	f.keyID = keyID
	_, f.ctxDeadline = ctx.Deadline()
	return []byte(strings.Repeat("a", 65)), nil
}

func (f *fakeBackend) SignTypedData(ctx context.Context, keyID string, domainHash, structHash [32]byte) ([32]byte, error) {
	f.keyID = keyID
	_, f.ctxDeadline = ctx.Deadline()
	var out [32]byte
	out[0] = 0x42
	return out, nil
}

func (f *fakeBackend) SignEIP712(ctx context.Context, keyID string, typed apitypes.TypedData) ([]byte, error) {
	f.keyID = keyID
	_, f.ctxDeadline = ctx.Deadline()
	return []byte(strings.Repeat("b", 65)), nil
}

func TestKMSSignerDelegatesWithTimeout(t *testing.T) {
	backend := &fakeBackend{}
	signer, err := New(Config{KeyID: "key-1", Address: "0xabc", ChainID: 137, Timeout: time.Second, Backend: backend})
	if err != nil {
		t.Fatal(err)
	}
	var _ signers.Signer = signer
	if signer.Address() != "0xabc" || signer.ChainID() != 137 {
		t.Fatalf("identity mismatch: %s %d", signer.Address(), signer.ChainID())
	}
	var hash [32]byte
	sig, err := signer.SignHash(hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 65 || backend.keyID != "key-1" || !backend.ctxDeadline {
		t.Fatalf("bad delegation sigLen=%d keyID=%q deadline=%v", len(sig), backend.keyID, backend.ctxDeadline)
	}
	result, err := signer.SignTypedData(hash, hash)
	if err != nil {
		t.Fatal(err)
	}
	if result[0] != 0x42 {
		t.Fatalf("unexpected typed-data result: %x", result)
	}
}

func TestKMSSignerRejectsMissingConfig(t *testing.T) {
	if _, err := New(Config{Backend: &fakeBackend{}}); err == nil {
		t.Fatal("expected missing key id error")
	}
	if _, err := New(Config{KeyID: "key-1"}); err == nil {
		t.Fatal("expected missing backend error")
	}
}
