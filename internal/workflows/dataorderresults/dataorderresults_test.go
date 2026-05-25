package dataorderresults

import (
	"context"
	"errors"
	"testing"

	sdkclob "github.com/TrebuchetDynamics/polygolem/pkg/clob"
	sdkorderresults "github.com/TrebuchetDynamics/polygolem/pkg/orderresults"
	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

func TestRunnerBuildsDataOnlyReportWithoutLoadingCLOBSecrets(t *testing.T) {
	data := &fakeDataReader{
		positions: []types.Position{{
			TokenID:      "token-up",
			ConditionID:  "0xmarket",
			Size:         2,
			AvgPrice:     0.4,
			CurrentPrice: 0.6,
			CurrentValue: 1.2,
			Outcome:      "Up",
			Title:        "BTC up",
		}},
	}
	privateKeyCalled := false
	runner := New(Config{
		Data: data,
		PrivateKey: func() (string, error) {
			privateKeyCalled = true
			return "", errors.New("private key should not be loaded")
		},
		CLOBFactory: func(sdkclob.Config) sdkorderresults.CLOBReader {
			t.Fatal("CLOB factory should not be called for data-only reports")
			return nil
		},
	})

	report, err := runner.Run(context.Background(), Request{User: " 0xwallet ", Limit: 7})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if privateKeyCalled {
		t.Fatal("private key loader was called for a data-only report")
	}
	if data.seenUser != "0xwallet" || data.seenLimit != 7 {
		t.Fatalf("data reader saw user=%q limit=%d", data.seenUser, data.seenLimit)
	}
	if report.User != "0xwallet" || report.Limit != 7 || report.Summary.Positions != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestRunnerIncludesAuthenticatedCLOBHistoryWithLoadedKeyAndCredentials(t *testing.T) {
	data := &fakeDataReader{}
	clob := &fakeCLOBReader{orders: []sdkclob.OrderRecord{{
		ID:           "order-live",
		Status:       "ORDER_STATUS_LIVE",
		Market:       "0xmarket",
		AssetID:      "token-up",
		Side:         "BUY",
		Price:        "0.5",
		OriginalSize: "2",
		Outcome:      "Up",
	}}}
	var gotConfig sdkclob.Config
	runner := New(Config{
		Data:        data,
		CLOBBaseURL: "https://clob.test",
		PrivateKey: func() (string, error) {
			return "0xprivate", nil
		},
		CLOBCredentials: func() (sdkclob.APIKey, bool) {
			return sdkclob.APIKey{Key: "key", Secret: "secret", Passphrase: "pass"}, true
		},
		CLOBFactory: func(cfg sdkclob.Config) sdkorderresults.CLOBReader {
			gotConfig = cfg
			return clob
		},
	})

	report, err := runner.Run(context.Background(), Request{User: "0xwallet", IncludeCLOB: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if gotConfig.BaseURL != "https://clob.test" {
		t.Fatalf("CLOB base URL=%q", gotConfig.BaseURL)
	}
	if gotConfig.Credentials.Key != "key" || gotConfig.Credentials.Secret != "secret" || gotConfig.Credentials.Passphrase != "pass" {
		t.Fatalf("CLOB credentials not adapted: %+v", gotConfig.Credentials)
	}
	if clob.seenPrivateKey != "0xprivate" {
		t.Fatalf("private key=%q", clob.seenPrivateKey)
	}
	if report.Summary.OpenOrders != 1 {
		t.Fatalf("open orders=%d, report=%+v", report.Summary.OpenOrders, report)
	}
}

func TestRunnerRequiresUserBeforeLoadingPrivateKey(t *testing.T) {
	privateKeyCalled := false
	runner := New(Config{
		Data: &fakeDataReader{},
		PrivateKey: func() (string, error) {
			privateKeyCalled = true
			return "0xprivate", nil
		},
	})

	_, err := runner.Run(context.Background(), Request{IncludeCLOB: true})
	if err == nil || err.Error() != "--user required" {
		t.Fatalf("error=%v, want --user required", err)
	}
	if privateKeyCalled {
		t.Fatal("private key loader was called before user validation")
	}
}

type fakeDataReader struct {
	positions []types.Position
	closed    []types.ClosedPosition
	trades    []types.Trade

	seenUser  string
	seenLimit int
}

func (f *fakeDataReader) CurrentPositionsWithLimit(_ context.Context, user string, limit int) ([]types.Position, error) {
	f.seenUser = user
	f.seenLimit = limit
	return f.positions, nil
}

func (f *fakeDataReader) ClosedPositionsWithLimit(context.Context, string, int) ([]types.ClosedPosition, error) {
	return f.closed, nil
}

func (f *fakeDataReader) Trades(context.Context, string, int) ([]types.Trade, error) {
	return f.trades, nil
}

type fakeCLOBReader struct {
	orders         []sdkclob.OrderRecord
	trades         []sdkclob.TradeRecord
	seenPrivateKey string
}

func (f *fakeCLOBReader) ListOrders(_ context.Context, privateKey string) ([]sdkclob.OrderRecord, error) {
	f.seenPrivateKey = privateKey
	return f.orders, nil
}

func (f *fakeCLOBReader) ListTrades(_ context.Context, privateKey string) ([]sdkclob.TradeRecord, error) {
	f.seenPrivateKey = privateKey
	return f.trades, nil
}
