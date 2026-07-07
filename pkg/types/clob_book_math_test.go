package types

import (
	"errors"
	"testing"
)

func mathBook(bids, asks []CLOBOrderBookLevel) CLOBOrderBook {
	return CLOBOrderBook{Bids: bids, Asks: asks}
}

func lvl(price, size string) CLOBOrderBookLevel {
	return CLOBOrderBookLevel{Price: price, Size: size}
}

func TestBestBidPicksHighestExecutableLevel(t *testing.T) {
	book := mathBook([]CLOBOrderBookLevel{
		lvl("0.41", "100"),
		lvl("0.44", "5"),
		lvl("0.43", "20"),
	}, nil)
	got, ok := book.BestBid()
	if !ok || got != 0.44 {
		t.Fatalf("BestBid = %v, %v; want 0.44, true", got, ok)
	}
}

func TestBestAskPicksLowestExecutableLevel(t *testing.T) {
	book := mathBook(nil, []CLOBOrderBookLevel{
		lvl("0.58", "9"),
		lvl("0.55", "3"),
		lvl("0.61", "50"),
	})
	got, ok := book.BestAsk()
	if !ok || got != 0.55 {
		t.Fatalf("BestAsk = %v, %v; want 0.55, true", got, ok)
	}
}

func TestBookMathSkipsUnexecutableLevels(t *testing.T) {
	book := mathBook(
		[]CLOBOrderBookLevel{
			lvl("", "10"),      // unparseable price
			lvl("0.50", ""),    // unparseable size
			lvl("0.50", "0"),   // zero size
			lvl("-0.10", "10"), // negative price
			lvl("1.20", "10"),  // above share price bound
		},
		[]CLOBOrderBookLevel{
			lvl("abc", "10"),
			lvl("0.30", "-4"),
		},
	)
	if _, ok := book.BestBid(); ok {
		t.Fatal("BestBid found a level in an unexecutable book")
	}
	if _, ok := book.BestAsk(); ok {
		t.Fatal("BestAsk found a level in an unexecutable book")
	}
}

func TestBestBidAskEmptyBook(t *testing.T) {
	var book CLOBOrderBook
	if _, ok := book.BestBid(); ok {
		t.Fatal("BestBid on empty book reported ok")
	}
	if _, ok := book.BestAsk(); ok {
		t.Fatal("BestAsk on empty book reported ok")
	}
}

func TestAvailableAskSizeSumsAtOrBelowMaxPrice(t *testing.T) {
	book := mathBook(nil, []CLOBOrderBookLevel{
		lvl("0.50", "10"),
		lvl("0.55", "20"),
		lvl("0.56", "40"), // above max, excluded
		lvl("bad", "5"),   // unparseable, excluded
	})
	if got := book.AvailableAskSize(0.55); got != 30 {
		t.Fatalf("AvailableAskSize(0.55) = %v; want 30", got)
	}
	if got := book.AvailableAskSize(0); got != 0 {
		t.Fatalf("AvailableAskSize(0) = %v; want 0", got)
	}
	if got := book.AvailableAskSize(-1); got != 0 {
		t.Fatalf("AvailableAskSize(-1) = %v; want 0", got)
	}
}

func TestTickSizeValue(t *testing.T) {
	cases := []struct {
		name    string
		in      CLOBTickSize
		want    float64
		wantErr bool
	}{
		{name: "tick size preferred", in: CLOBTickSize{TickSize: "0.01", MinimumTickSize: "0.001"}, want: 0.01},
		{name: "falls back to minimum tick size", in: CLOBTickSize{MinimumTickSize: "0.001"}, want: 0.001},
		{name: "whitespace tolerated", in: CLOBTickSize{TickSize: " 0.1 "}, want: 0.1},
		{name: "zero rejected", in: CLOBTickSize{TickSize: "0"}, wantErr: true},
		{name: "empty payload", in: CLOBTickSize{}, wantErr: true},
		{name: "garbage rejected", in: CLOBTickSize{TickSize: "abc", MinimumTickSize: "-1"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.in.Value()
			if tc.wantErr {
				if !errors.Is(err, ErrTickSizeUnavailable) {
					t.Fatalf("Value() err = %v; want ErrTickSizeUnavailable", err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("Value() = %v, %v; want %v, nil", got, err, tc.want)
			}
		})
	}
}
