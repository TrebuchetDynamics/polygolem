package clobaccountreads

import (
	"context"
	"errors"
	"testing"

	internalclob "github.com/TrebuchetDynamics/polygolem/internal/clob"
)

func TestRunnerOrdersValidatesOutputBeforeLoadingPrivateKey(t *testing.T) {
	privateKeyCalled := false
	runner := New(Config{
		Reader: &fakeReader{},
		PrivateKey: func() (string, error) {
			privateKeyCalled = true
			return "0xprivate", nil
		},
	})

	_, err := runner.Orders(context.Background(), Request{Output: "table"})
	if err == nil || err.Error() != "only --output json is supported" {
		t.Fatalf("error=%v, want only --output json is supported", err)
	}
	if privateKeyCalled {
		t.Fatal("private key loader was called before output validation")
	}
}

func TestRunnerReadsOrdersOrderAndTradesWithLoadedPrivateKey(t *testing.T) {
	reader := &fakeReader{
		orders: []internalclob.OrderRecord{{ID: "order-1"}},
		order:  &internalclob.OrderRecord{ID: "order-2"},
		trades: []internalclob.TradeRecord{{ID: "trade-1"}},
	}
	runner := New(Config{
		Reader: reader,
		PrivateKey: func() (string, error) {
			return "0xprivate", nil
		},
	})

	orders, err := runner.Orders(context.Background(), Request{Output: "json"})
	if err != nil || len(orders) != 1 || orders[0].ID != "order-1" {
		t.Fatalf("Orders result=%+v err=%v", orders, err)
	}
	if reader.ordersPrivateKey != "0xprivate" {
		t.Fatalf("orders private key=%q", reader.ordersPrivateKey)
	}

	order, err := runner.Order(context.Background(), OrderRequest{OrderID: "order-2"})
	if err != nil || order.ID != "order-2" {
		t.Fatalf("Order result=%+v err=%v", order, err)
	}
	if reader.orderPrivateKey != "0xprivate" || reader.orderID != "order-2" {
		t.Fatalf("order private key=%q orderID=%q", reader.orderPrivateKey, reader.orderID)
	}

	trades, err := runner.Trades(context.Background(), Request{})
	if err != nil || len(trades) != 1 || trades[0].ID != "trade-1" {
		t.Fatalf("Trades result=%+v err=%v", trades, err)
	}
	if reader.tradesPrivateKey != "0xprivate" {
		t.Fatalf("trades private key=%q", reader.tradesPrivateKey)
	}
}

func TestRunnerPropagatesPrivateKeyAndReaderErrors(t *testing.T) {
	wantKeyErr := errors.New("missing private key")
	runner := New(Config{Reader: &fakeReader{}, PrivateKey: func() (string, error) { return "", wantKeyErr }})
	_, err := runner.Trades(context.Background(), Request{})
	if !errors.Is(err, wantKeyErr) {
		t.Fatalf("private key error=%v, want %v", err, wantKeyErr)
	}

	wantReaderErr := errors.New("clob down")
	runner = New(Config{Reader: &fakeReader{err: wantReaderErr}, PrivateKey: func() (string, error) { return "0xprivate", nil }})
	_, err = runner.Order(context.Background(), OrderRequest{OrderID: "order-1"})
	if !errors.Is(err, wantReaderErr) {
		t.Fatalf("reader error=%v, want %v", err, wantReaderErr)
	}
}

type fakeReader struct {
	orders []internalclob.OrderRecord
	order  *internalclob.OrderRecord
	trades []internalclob.TradeRecord
	err    error

	ordersPrivateKey string
	orderPrivateKey  string
	orderID          string
	tradesPrivateKey string
}

func (f *fakeReader) ListOrders(_ context.Context, privateKey string) ([]internalclob.OrderRecord, error) {
	f.ordersPrivateKey = privateKey
	return f.orders, f.err
}

func (f *fakeReader) Order(_ context.Context, privateKey, orderID string) (*internalclob.OrderRecord, error) {
	f.orderPrivateKey = privateKey
	f.orderID = orderID
	return f.order, f.err
}

func (f *fakeReader) ListTrades(_ context.Context, privateKey string) ([]internalclob.TradeRecord, error) {
	f.tradesPrivateKey = privateKey
	return f.trades, f.err
}
