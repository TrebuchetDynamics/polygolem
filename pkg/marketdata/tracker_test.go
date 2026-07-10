package marketdata

import (
	"testing"

	"github.com/TrebuchetDynamics/polygolem/pkg/stream"
)

func TestTrackerBookComputesBestBidAskAndMidpoint(t *testing.T) {
	tracker := NewTracker()

	snapshot := tracker.ApplyBook(stream.BookMessage{
		EventType: "book",
		AssetID:   "token-1",
		Market:    "market-1",
		Timestamp: "1000",
		Bids: []stream.PriceLevel{
			{Price: "0.49", Size: "10"},
			{Price: "0.51", Size: "4"},
		},
		Asks: []stream.PriceLevel{
			{Price: "0.55", Size: "2"},
			{Price: "0.53", Size: "7"},
		},
	})

	if snapshot.AssetID != "token-1" || snapshot.Market != "market-1" {
		t.Fatalf("unexpected identity fields: %+v", snapshot)
	}
	if snapshot.BestBid != "0.51" || snapshot.BestAsk != "0.53" || snapshot.Midpoint != "0.52" {
		t.Fatalf("prices bid=%q ask=%q midpoint=%q", snapshot.BestBid, snapshot.BestAsk, snapshot.Midpoint)
	}
	if len(snapshot.Bids) != 2 || snapshot.Bids[0].Price != "0.51" {
		t.Fatalf("bids not sorted best-first: %+v", snapshot.Bids)
	}
	if len(snapshot.Asks) != 2 || snapshot.Asks[0].Price != "0.53" {
		t.Fatalf("asks not sorted best-first: %+v", snapshot.Asks)
	}
}

func TestTrackerPriceChangeUpdatesBookAndBestPrices(t *testing.T) {
	tracker := NewTracker()
	tracker.ApplyBook(stream.BookMessage{
		EventType: "book",
		AssetID:   "token-1",
		Market:    "market-1",
		Bids:      []stream.PriceLevel{{Price: "0.49", Size: "10"}},
		Asks:      []stream.PriceLevel{{Price: "0.53", Size: "7"}},
	})

	snapshots := tracker.ApplyPriceChange(stream.PriceChangeMessage{
		EventType: "price_change",
		Market:    "market-1",
		Timestamp: "1001",
		PriceChanges: []stream.PriceChangeEntry{{
			AssetID: "token-1",
			Side:    "BUY",
			Price:   "0.52",
			Size:    "12",
			BestBid: "0.52",
			BestAsk: "0.53",
			Hash:    "hash-1",
		}},
	})

	if len(snapshots) != 1 {
		t.Fatalf("snapshot count=%d, want 1", len(snapshots))
	}
	snapshot := snapshots[0]
	if snapshot.EventType != "price_change" || snapshot.UpdateHash != "hash-1" {
		t.Fatalf("unexpected event metadata: %+v", snapshot)
	}
	if snapshot.BestBid != "0.52" || snapshot.BestAsk != "0.53" || snapshot.Midpoint != "0.525" {
		t.Fatalf("prices bid=%q ask=%q midpoint=%q", snapshot.BestBid, snapshot.BestAsk, snapshot.Midpoint)
	}
	if len(snapshot.Bids) == 0 || snapshot.Bids[0].Price != "0.52" || snapshot.Bids[0].Size != "12" {
		t.Fatalf("bid level not updated: %+v", snapshot.Bids)
	}
}

func TestTrackerLastTradePreservesBookPrices(t *testing.T) {
	tracker := NewTracker()
	tracker.ApplyBook(stream.BookMessage{
		EventType: "book",
		AssetID:   "token-1",
		Market:    "market-1",
		Bids:      []stream.PriceLevel{{Price: "0.49", Size: "10"}},
		Asks:      []stream.PriceLevel{{Price: "0.53", Size: "7"}},
	})

	snapshot := tracker.ApplyLastTrade(stream.LastTradeMessage{
		EventType:       "last_trade_price",
		AssetID:         "token-1",
		Market:          "market-1",
		Price:           "0.5",
		Size:            "25",
		Side:            "BUY",
		Timestamp:       "1002",
		TransactionHash: "0xabc",
	})

	if snapshot.LastTradePrice != "0.5" || snapshot.LastTradeSize != "25" || snapshot.LastTradeSide != "BUY" {
		t.Fatalf("last trade not tracked: %+v", snapshot)
	}
	if snapshot.BestBid != "0.49" || snapshot.BestAsk != "0.53" || snapshot.Midpoint != "0.51" {
		t.Fatalf("book prices not preserved: %+v", snapshot)
	}
	if snapshot.TransactionHash != "0xabc" {
		t.Fatalf("transaction hash=%q, want 0xabc", snapshot.TransactionHash)
	}
}

func TestTrackerPriceChangeBestBidAskOverrideStaleBookLevels(t *testing.T) {
	tracker := NewTracker()
	tracker.ApplyBook(stream.BookMessage{
		EventType: "book",
		AssetID:   "token-1",
		Market:    "market-1",
		Bids:      []stream.PriceLevel{{Price: "0.49", Size: "10"}},
		Asks:      []stream.PriceLevel{{Price: "0.53", Size: "7"}},
	})

	snapshots := tracker.ApplyPriceChange(stream.PriceChangeMessage{
		EventType: "price_change",
		Market:    "market-1",
		Timestamp: "1003",
		PriceChanges: []stream.PriceChangeEntry{{
			AssetID: "token-1",
			BestBid: "0.50",
			BestAsk: "0.54",
		}},
	})

	if len(snapshots) != 1 {
		t.Fatalf("snapshot count=%d, want 1", len(snapshots))
	}
	if snapshots[0].BestBid != "0.50" || snapshots[0].BestAsk != "0.54" || snapshots[0].Midpoint != "0.52" {
		t.Fatalf("best prices did not override stale levels: %+v", snapshots[0])
	}
}

func TestTrackerBestBidAskUpdatesSnapshotWithoutBookDelta(t *testing.T) {
	tracker := NewTracker()

	snapshot := tracker.ApplyBestBidAsk(stream.BestBidAskMessage{
		EventType: "best_bid_ask",
		AssetID:   "token-1",
		Market:    "market-1",
		BestBid:   "0.73",
		BestAsk:   "0.77",
		Spread:    "0.04",
		Timestamp: "1004",
	})

	if snapshot.EventType != "best_bid_ask" || snapshot.AssetID != "token-1" {
		t.Fatalf("unexpected identity: %+v", snapshot)
	}
	if snapshot.BestBid != "0.73" || snapshot.BestAsk != "0.77" || snapshot.Spread != "0.04" || snapshot.Midpoint != "0.75" {
		t.Fatalf("unexpected best bid/ask snapshot: %+v", snapshot)
	}
}

func TestLiquidityAndDepthForSnapshot(t *testing.T) {
	snapshot := Snapshot{
		Market:  "market-1",
		AssetID: "token-1",
		Bids: []Level{
			{Price: "0.49", Size: "10"},
			{Price: "0.47", Size: "5"},
		},
		Asks: []Level{
			{Price: "0.51", Size: "20"},
			{Price: "0.54", Size: "7"},
		},
	}

	liquidity := LiquidityFor(snapshot)
	if liquidity.BidSize != 10 || liquidity.AskSize != 20 || liquidity.Imbalance != 1.0/3.0 {
		t.Fatalf("unexpected liquidity: %#v", liquidity)
	}

	depth := DepthFor(snapshot)
	if depth.TotalBid != 15 || depth.TotalAsk != 27 {
		t.Fatalf("unexpected totals: %#v", depth)
	}
	if depth.BidDepth1C != 10 || depth.BidDepth2C != 15 || depth.BidDepth5C != 15 {
		t.Fatalf("unexpected bid depth: %#v", depth)
	}
	if depth.AskDepth1C != 20 || depth.AskDepth2C != 20 || depth.AskDepth5C != 27 {
		t.Fatalf("unexpected ask depth: %#v", depth)
	}
}

func TestTrackerTickSizeChangeTracksCurrentTickSize(t *testing.T) {
	tracker := NewTracker()

	snapshot := tracker.ApplyTickSizeChange(stream.TickSizeChangeMessage{
		EventType:   "tick_size_change",
		AssetID:     "token-1",
		Market:      "market-1",
		OldTickSize: "0.01",
		NewTickSize: "0.001",
		Timestamp:   "1005",
	})

	if snapshot.EventType != "tick_size_change" || snapshot.TickSize != "0.001" || snapshot.PreviousTickSize != "0.01" {
		t.Fatalf("unexpected tick-size snapshot: %+v", snapshot)
	}
}
