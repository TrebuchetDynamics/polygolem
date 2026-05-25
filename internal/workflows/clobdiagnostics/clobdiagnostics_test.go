package clobdiagnostics

import (
	"context"
	"errors"
	"testing"

	internalclob "github.com/TrebuchetDynamics/polygolem/internal/clob"
)

func TestRunnerListBuilderFeeKeysValidatesOutputBeforeLoadingPrivateKey(t *testing.T) {
	privateKeyCalled := false
	runner := New(Config{
		Reader: &fakeReader{},
		PrivateKey: func() (string, error) {
			privateKeyCalled = true
			return "0xprivate", nil
		},
	})

	_, err := runner.ListBuilderFeeKeys(context.Background(), Request{Output: "table"})
	if err == nil || err.Error() != "only --output json is supported" {
		t.Fatalf("error=%v, want only --output json is supported", err)
	}
	if privateKeyCalled {
		t.Fatal("private key loader was called before output validation")
	}
}

func TestRunnerListsBuilderFeeKeysAndRunsMarketTradesProbe(t *testing.T) {
	reader := &fakeReader{
		feeKeys: []internalclob.BuilderFeeKeyRecord{{Key: "builder-key"}},
		probe: &internalclob.MarketTradesProbeResult{
			Classification: internalclob.ProbeMarketWide,
			SelectorType:   "market",
			Selector:       "0xmarket",
			RowCount:       2,
		},
	}
	runner := New(Config{
		Reader: reader,
		PrivateKey: func() (string, error) {
			return "0xprivate", nil
		},
	})

	feeKeys, err := runner.ListBuilderFeeKeys(context.Background(), Request{Output: "json"})
	if err != nil || len(feeKeys) != 1 || feeKeys[0].Key != "builder-key" {
		t.Fatalf("ListBuilderFeeKeys result=%+v err=%v", feeKeys, err)
	}
	if reader.feeKeysPrivateKey != "0xprivate" {
		t.Fatalf("fee key private key=%q", reader.feeKeysPrivateKey)
	}

	probe, err := runner.MarketTradesProbe(context.Background(), ProbeRequest{
		Market:     "0xmarket",
		AssetID:    "token-1",
		NextCursor: "cursor-1",
	})
	if err != nil || probe.RowCount != 2 || probe.Classification != internalclob.ProbeMarketWide {
		t.Fatalf("MarketTradesProbe result=%+v err=%v", probe, err)
	}
	if reader.probePrivateKey != "0xprivate" {
		t.Fatalf("probe private key=%q", reader.probePrivateKey)
	}
	if reader.probeParams.Market != "0xmarket" || reader.probeParams.AssetID != "token-1" || reader.probeParams.NextCursor != "cursor-1" {
		t.Fatalf("probe params=%+v", reader.probeParams)
	}
}

func TestRunnerPropagatesPrivateKeyAndReaderErrors(t *testing.T) {
	wantKeyErr := errors.New("missing private key")
	runner := New(Config{Reader: &fakeReader{}, PrivateKey: func() (string, error) { return "", wantKeyErr }})
	_, err := runner.MarketTradesProbe(context.Background(), ProbeRequest{})
	if !errors.Is(err, wantKeyErr) {
		t.Fatalf("private key error=%v, want %v", err, wantKeyErr)
	}

	wantReaderErr := errors.New("clob down")
	runner = New(Config{Reader: &fakeReader{err: wantReaderErr}, PrivateKey: func() (string, error) { return "0xprivate", nil }})
	_, err = runner.ListBuilderFeeKeys(context.Background(), Request{})
	if !errors.Is(err, wantReaderErr) {
		t.Fatalf("reader error=%v, want %v", err, wantReaderErr)
	}
}

type fakeReader struct {
	feeKeys []internalclob.BuilderFeeKeyRecord
	probe   *internalclob.MarketTradesProbeResult
	err     error

	feeKeysPrivateKey string
	probePrivateKey   string
	probeParams       internalclob.MarketTradesProbeRequest
}

func (f *fakeReader) ListBuilderFeeKeys(_ context.Context, privateKey string) ([]internalclob.BuilderFeeKeyRecord, error) {
	f.feeKeysPrivateKey = privateKey
	return f.feeKeys, f.err
}

func (f *fakeReader) MarketTradesProbe(_ context.Context, privateKey string, params internalclob.MarketTradesProbeRequest) (*internalclob.MarketTradesProbeResult, error) {
	f.probePrivateKey = privateKey
	f.probeParams = params
	return f.probe, f.err
}
