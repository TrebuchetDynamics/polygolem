package tests

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	clobsdk "github.com/TrebuchetDynamics/polygolem/pkg/clob"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
	"github.com/TrebuchetDynamics/polygolem/pkg/types"
	"github.com/TrebuchetDynamics/polygolem/pkg/universal"
)

func TestBTCFiveMinuteLiveCLOBContracts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live BTC 5m CLOB contract test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	resolved := resolveLiveBTC5m(t, ctx)
	client := universal.NewClient(universal.Config{})
	tokens := []string{resolved.UpTokenID, resolved.DownTokenID}
	if resolved.UpTokenID == resolved.DownTokenID {
		t.Fatalf("resolved identical up/down token IDs: %s", resolved.UpTokenID)
	}
	t.Logf("BTC 5m %s condition=%s up=%s down=%s", resolved.StartDate.Format(time.RFC3339), resolved.ConditionID, resolved.UpTokenID, resolved.DownTokenID)

	slug := marketresolver.CryptoWindowSlug("BTC", "5m", resolved.StartDate)
	event, err := client.EventBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("EventBySlug(%s): %v", slug, err)
	}
	gammaMarket := gammaMarketForCondition(t, event.Markets, resolved.ConditionID)
	if event.Slug != slug {
		t.Fatalf("Gamma event slug=%q want %q", event.Slug, slug)
	}
	if !event.Active || event.Closed || gammaMarket.Closed || !gammaMarket.Active || !gammaMarket.EnableOrderBook || !gammaMarket.AcceptingOrders {
		t.Fatalf("Gamma BTC 5m market is not active/orderbook-backed: event=%+v market=%+v", event, gammaMarket)
	}
	assertGammaFiveMinuteWindow(t, gammaMarket, resolved.StartDate)
	if outcomes := []string(gammaMarket.Outcomes); len(outcomes) != 2 {
		t.Fatalf("Gamma outcomes=%v want 2 outcomes", outcomes)
	}
	for _, tokenID := range tokens {
		if !strings.Contains(gammaMarket.ClobTokenIDs, tokenID) {
			t.Fatalf("Gamma clobTokenIds %q does not include resolved token %s", gammaMarket.ClobTokenIDs, tokenID)
		}
	}

	market, err := client.CLOBMarket(ctx, resolved.ConditionID)
	if err != nil {
		t.Fatalf("CLOBMarket(%s): %v", resolved.ConditionID, err)
	}
	if market.ConditionID != resolved.ConditionID {
		t.Fatalf("CLOB market condition_id=%q want %q", market.ConditionID, resolved.ConditionID)
	}
	if market.Closed || market.Archived || !market.EnableOrderBook || !market.AcceptingOrders {
		t.Fatalf("CLOB market not tradeable/readable enough for active BTC 5m contract: %+v", market)
	}
	assertUnitDecimal(t, "CLOB market spread", fmt.Sprint(market.Spread))
	assertNonNegativeBps(t, "CLOB market maker_base_fee", market.MakerBaseFee)
	assertNonNegativeBps(t, "CLOB market taker_base_fee", market.TakerBaseFee)
	assertPositiveIfSet(t, "CLOB market order_min_size", market.OrderMinSize)
	assertPositiveIfSet(t, "CLOB market order_price_min_tick_size", market.OrderPriceMinTickSize)
	if market.RewardsMinSize < 0 || market.RewardsMaxSpread < 0 || market.MinimumOrderAge < 0 {
		t.Fatalf("CLOB market reward/age fields must be non-negative: %+v", market)
	}
	if len(market.Tokens) != 2 {
		t.Fatalf("CLOB market tokens=%d want 2: %+v", len(market.Tokens), market.Tokens)
	}
	for _, tokenID := range tokens {
		if !clobMarketContainsToken(market.Tokens, tokenID) {
			t.Fatalf("CLOB market tokens do not include resolved token %s: %+v", tokenID, market.Tokens)
		}
	}
	marketTokenPrices := map[string]float64{}
	for _, token := range market.Tokens {
		if token.TokenID == "" || token.Outcome == "" {
			t.Fatalf("CLOB market token missing identity: %+v", token)
		}
		if !containsOutcome([]string(gammaMarket.Outcomes), token.Outcome) {
			t.Fatalf("CLOB token outcome %q not in Gamma outcomes %v", token.Outcome, []string(gammaMarket.Outcomes))
		}
		marketTokenPrices[token.TokenID] = parseUnitDecimal(t, "CLOB market token price", token.Price)
	}
	if tokenPriceSum := marketTokenPrices[resolved.UpTokenID] + marketTokenPrices[resolved.DownTokenID]; tokenPriceSum < 0.90 || tokenPriceSum > 1.10 {
		t.Fatalf("CLOB market up/down token price sum=%f outside [0.90,1.10]", tokenPriceSum)
	}

	outcome, err := clobsdk.NewClient(clobsdk.DefaultConfig()).MarketOutcome(ctx, resolved.ConditionID, "")
	if err != nil {
		t.Fatalf("MarketOutcome(%s): %v", resolved.ConditionID, err)
	}
	if outcome.Status != types.CLOBOutcomeUnresolved || outcome.ConditionID != resolved.ConditionID || outcome.Closed || outcome.WinningTokenID != "" {
		t.Fatalf("active BTC 5m outcome should be unresolved without winner: %+v", outcome)
	}
	if !strings.Contains(outcome.Source, "clob:/markets/"+resolved.ConditionID) {
		t.Fatalf("MarketOutcome source=%q missing CLOB condition", outcome.Source)
	}

	buyPriceValues := map[string]float64{}
	sellPriceValues := map[string]float64{}
	midpointValues := map[string]float64{}
	byTokenUp, err := client.CLOBMarketByToken(ctx, resolved.UpTokenID)
	if err != nil {
		t.Fatalf("CLOBMarketByToken(up): %v", err)
	}
	byTokenDown, err := client.CLOBMarketByToken(ctx, resolved.DownTokenID)
	if err != nil {
		t.Fatalf("CLOBMarketByToken(down): %v", err)
	}
	assertMarketByTokenPair(t, byTokenUp, resolved)
	assertMarketByTokenPair(t, byTokenDown, resolved)
	if byTokenUp.PrimaryTokenID != byTokenDown.PrimaryTokenID || byTokenUp.SecondaryTokenID != byTokenDown.SecondaryTokenID {
		t.Fatalf("markets-by-token pair mismatch: up=%+v down=%+v", byTokenUp, byTokenDown)
	}

	for _, tokenID := range tokens {
		t.Run(shortToken(tokenID), func(t *testing.T) {
			byToken, err := client.CLOBMarketByToken(ctx, tokenID)
			if err != nil {
				t.Fatalf("CLOBMarketByToken(%s): %v", tokenID, err)
			}
			if byToken.ConditionID != resolved.ConditionID {
				t.Fatalf("markets-by-token condition_id=%q want %q", byToken.ConditionID, resolved.ConditionID)
			}
			if byToken.PrimaryTokenID != tokenID && byToken.SecondaryTokenID != tokenID {
				t.Fatalf("markets-by-token response does not reference token %s: %+v", tokenID, byToken)
			}

			book, err := client.OrderBook(ctx, tokenID)
			if err != nil {
				t.Fatalf("OrderBook(%s): %v", tokenID, err)
			}
			if book.Market != resolved.ConditionID || book.AssetID != tokenID {
				t.Fatalf("book identity market=%q asset=%q want market=%q asset=%q", book.Market, book.AssetID, resolved.ConditionID, tokenID)
			}
			assertBookMetadata(t, book, market.NegRisk)
			if len(book.Bids)+len(book.Asks) == 0 {
				t.Fatalf("book has no bids or asks: %+v", book)
			}
			var bestBid, bestAsk float64
			if len(book.Bids) > 0 {
				bestBid = bestBookPrice(t, "bid", book.Bids, true)
			}
			if len(book.Asks) > 0 {
				bestAsk = bestBookPrice(t, "ask", book.Asks, false)
			}
			assertPositiveDecimal(t, "book.tick_size", book.TickSize)
			assertPositiveDecimal(t, "book.min_order_size", book.MinOrderSize)

			buy, err := client.Price(ctx, tokenID, "BUY")
			if err != nil {
				t.Fatalf("Price BUY(%s): %v", tokenID, err)
			}
			sell, err := client.Price(ctx, tokenID, "SELL")
			if err != nil {
				t.Fatalf("Price SELL(%s): %v", tokenID, err)
			}
			midpoint, err := client.Midpoint(ctx, tokenID)
			if err != nil {
				t.Fatalf("Midpoint(%s): %v", tokenID, err)
			}
			spread, err := client.Spread(ctx, tokenID)
			if err != nil {
				t.Fatalf("Spread(%s): %v", tokenID, err)
			}
			buyPriceValues[tokenID] = parseUnitDecimal(t, "price BUY", buy)
			sellPriceValues[tokenID] = parseUnitDecimal(t, "price SELL", sell)
			midpointFloat := parseUnitDecimal(t, "midpoint", midpoint)
			midpointValues[tokenID] = midpointFloat
			if len(book.Bids) > 0 && len(book.Asks) > 0 {
				if bestBid > bestAsk {
					t.Fatalf("crossed book best_bid=%f best_ask=%f", bestBid, bestAsk)
				}
				if midpointFloat < bestBid-0.05 || midpointFloat > bestAsk+0.05 {
					t.Fatalf("midpoint=%f too far outside book best bid/ask [%f,%f]", midpointFloat, bestBid, bestAsk)
				}
			}
			spreadFloat := parseDecimal(t, "spread", spread)
			if spreadFloat < 0 || spreadFloat > 1 {
				t.Fatalf("spread=%f outside [0,1]", spreadFloat)
			}
			if len(book.Bids) > 0 && len(book.Asks) > 0 && math.Abs(spreadFloat-(bestAsk-bestBid)) > 0.02 {
				t.Logf("spread=%f differs from non-atomic book snapshot bid/ask [%f,%f]", spreadFloat, bestBid, bestAsk)
			}

			tick, err := client.TickSize(ctx, tokenID)
			if err != nil {
				t.Fatalf("TickSize(%s): %v", tokenID, err)
			}
			assertPositiveDecimal(t, "tick size", firstNonEmpty(tick.MinimumTickSize, tick.TickSize))

			fee, err := client.FeeRateBps(ctx, tokenID)
			if err != nil {
				t.Fatalf("FeeRateBps(%s): %v", tokenID, err)
			}
			if fee < 0 || fee > 10_000 {
				t.Fatalf("fee bps=%d outside [0,10000]", fee)
			}
			negRisk, err := client.NegRisk(ctx, tokenID)
			if err != nil {
				t.Fatalf("NegRisk(%s): %v", tokenID, err)
			}
			if negRisk.NegRisk != market.NegRisk {
				t.Fatalf("token neg_risk=%v want market neg_risk=%v", negRisk.NegRisk, market.NegRisk)
			}
			if negRisk.NegRiskFeeBips < 0 || negRisk.NegRiskFeeBips > 10_000 {
				t.Fatalf("neg risk fee bips=%d outside [0,10000]", negRisk.NegRiskFeeBips)
			}
		})
	}

	books, err := client.OrderBooks(ctx, []types.CLOBBookParams{{TokenID: resolved.UpTokenID}, {TokenID: resolved.DownTokenID}})
	if err != nil {
		t.Fatalf("OrderBooks: %v", err)
	}
	if len(books) != 2 {
		t.Fatalf("OrderBooks returned %d books, want 2", len(books))
	}
	for _, book := range books {
		if book.Market != resolved.ConditionID || (book.AssetID != resolved.UpTokenID && book.AssetID != resolved.DownTokenID) {
			t.Fatalf("batch book identity mismatch: %+v", book)
		}
		assertBookMetadata(t, &book, market.NegRisk)
		if len(book.Bids)+len(book.Asks) == 0 {
			t.Fatalf("batch book has no bids or asks: %+v", book)
		}
		assertPositiveDecimal(t, "batch book tick_size", book.TickSize)
		assertPositiveDecimal(t, "batch book min_order_size", book.MinOrderSize)
	}

	buyParams := []types.CLOBBookParams{{TokenID: resolved.UpTokenID, Side: "BUY"}, {TokenID: resolved.DownTokenID, Side: "BUY"}}
	sellParams := []types.CLOBBookParams{{TokenID: resolved.UpTokenID, Side: "SELL"}, {TokenID: resolved.DownTokenID, Side: "SELL"}}
	prices, err := client.Prices(ctx, buyParams)
	if err != nil {
		t.Fatalf("Prices BUY batch: %v", err)
	}
	sellPrices, err := client.Prices(ctx, sellParams)
	if err != nil {
		t.Fatalf("Prices SELL batch: %v", err)
	}
	midpoints, err := client.Midpoints(ctx, buyParams)
	if err != nil {
		t.Fatalf("Midpoints batch: %v", err)
	}
	for _, tokenID := range tokens {
		batchPrice := parseUnitDecimal(t, "batch BUY price "+shortToken(tokenID), prices[tokenID])
		batchSellPrice := parseUnitDecimal(t, "batch SELL price "+shortToken(tokenID), sellPrices[tokenID])
		batchMidpoint := parseUnitDecimal(t, "batch midpoint "+shortToken(tokenID), midpoints[tokenID])
		_ = batchPrice
		_ = batchSellPrice
		_ = batchMidpoint
	}
	lastTrades, err := client.LastTradesPrices(ctx, buyParams)
	if err != nil {
		t.Fatalf("LastTradesPrices batch: %v", err)
	}
	for _, tokenID := range tokens {
		lastTrade, err := client.LastTradePrice(ctx, tokenID)
		if err != nil {
			t.Fatalf("LastTradePrice(%s): %v", tokenID, err)
		}
		parseUnitDecimal(t, "last trade "+shortToken(tokenID), lastTrade)
		parseUnitDecimal(t, "batch last trade "+shortToken(tokenID), lastTrades[tokenID])
	}

	midpointSum := parseDecimal(t, "batch up midpoint", midpoints[resolved.UpTokenID]) + parseDecimal(t, "batch down midpoint", midpoints[resolved.DownTokenID])
	if midpointSum < 0.90 || midpointSum > 1.10 {
		t.Fatalf("BTC 5m up/down midpoint sum=%f outside [0.90,1.10]", midpointSum)
	}
	if len(midpointValues) != 2 {
		t.Fatalf("single midpoint checks ran for %d tokens, want 2", len(midpointValues))
	}

	for _, tokenID := range tokens {
		history, err := client.PricesHistory(ctx, &types.CLOBPriceHistoryParams{Market: tokenID, Interval: "1m", Fidelity: 10})
		if err != nil {
			t.Fatalf("PricesHistory(%s): %v", tokenID, err)
		}
		if history == nil || len(history.History) == 0 {
			t.Fatalf("price history empty for BTC 5m token %s", tokenID)
		}
		lastHistoryTS := int64(0)
		for i, point := range history.History {
			if point.T == "" {
				t.Fatalf("history[%s][%d] missing timestamp: %+v", shortToken(tokenID), i, point)
			}
			ts, err := strconv.ParseInt(point.T, 10, 64)
			if err != nil || ts <= 0 {
				t.Fatalf("history[%s][%d] bad timestamp %q: %v", shortToken(tokenID), i, point.T, err)
			}
			if ts < lastHistoryTS {
				t.Fatalf("history[%s] timestamps not sorted at %d: %d before %d", shortToken(tokenID), i, ts, lastHistoryTS)
			}
			lastHistoryTS = ts
			assertUnitDecimal(t, fmt.Sprintf("history[%s][%d].price", shortToken(tokenID), i), point.P)
		}
	}
}

func assertGammaFiveMinuteWindow(t *testing.T, market types.Market, wantStart time.Time) {
	t.Helper()
	start := market.EventStartTime.Time()
	if start.IsZero() {
		start = market.StartDate.Time()
	}
	if start.IsZero() {
		t.Fatalf("Gamma market missing start time: %+v", market)
	}
	if !start.UTC().Truncate(time.Second).Equal(wantStart.UTC().Truncate(time.Second)) {
		t.Fatalf("Gamma market start=%s want %s", start.UTC().Format(time.RFC3339), wantStart.UTC().Format(time.RFC3339))
	}
	end := market.EndDate.Time()
	if end.IsZero() {
		t.Fatalf("Gamma market missing end time: %+v", market)
	}
	if d := end.Sub(start); d != 5*time.Minute {
		t.Fatalf("Gamma market window duration=%s want 5m", d)
	}
}

func assertMarketByTokenPair(t *testing.T, got *types.CLOBMarketByTokenResponse, resolved marketresolver.ResolveResult) {
	t.Helper()
	if got.ConditionID != resolved.ConditionID {
		t.Fatalf("markets-by-token condition_id=%q want %q", got.ConditionID, resolved.ConditionID)
	}
	if got.PrimaryTokenID == got.SecondaryTokenID || got.PrimaryTokenID == "" || got.SecondaryTokenID == "" {
		t.Fatalf("bad markets-by-token pair: %+v", got)
	}
	if !((got.PrimaryTokenID == resolved.UpTokenID && got.SecondaryTokenID == resolved.DownTokenID) || (got.PrimaryTokenID == resolved.DownTokenID && got.SecondaryTokenID == resolved.UpTokenID)) {
		t.Fatalf("markets-by-token pair %+v does not match resolved up/down", got)
	}
}

func containsOutcome(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(value, needle) {
			return true
		}
	}
	return false
}

func gammaMarketForCondition(t *testing.T, markets []types.Market, conditionID string) types.Market {
	t.Helper()
	for _, market := range markets {
		if market.ConditionID == conditionID {
			return market
		}
	}
	t.Fatalf("Gamma event does not include condition %s", conditionID)
	return types.Market{}
}

func assertBookMetadata(t *testing.T, book *types.CLOBOrderBook, wantNegRisk bool) {
	t.Helper()
	if book.Hash == "" {
		t.Fatalf("book hash is empty: %+v", book)
	}
	if ts, err := strconv.ParseInt(book.Timestamp, 10, 64); err != nil || ts <= 0 {
		t.Fatalf("book timestamp=%q invalid: %v", book.Timestamp, err)
	}
	if book.NegRisk != wantNegRisk {
		t.Fatalf("book neg_risk=%v want market neg_risk=%v", book.NegRisk, wantNegRisk)
	}
	if book.LastTradePrice != "" {
		assertUnitDecimal(t, "book last_trade_price", book.LastTradePrice)
	}
}

func bestBookPrice(t *testing.T, side string, levels []types.CLOBOrderBookLevel, max bool) float64 {
	t.Helper()
	if len(levels) == 0 {
		t.Fatalf("%s side has no levels", side)
	}
	best := parseDecimal(t, side+" price", levels[0].Price)
	assertPositiveDecimal(t, side+" size", levels[0].Size)
	for _, level := range levels[1:] {
		price := parseDecimal(t, side+" price", level.Price)
		assertPositiveDecimal(t, side+" size", level.Size)
		if max && price > best {
			best = price
		}
		if !max && price < best {
			best = price
		}
	}
	return best
}

func resolveLiveBTC5m(t *testing.T, ctx context.Context) marketresolver.ResolveResult {
	t.Helper()
	now := time.Now().UTC()
	window := time.Unix(now.Unix()-(now.Unix()%300), 0).UTC()
	resolver := marketresolver.NewResolver("")

	var attempts []string
	for _, candidate := range []time.Time{window, window.Add(-5 * time.Minute)} {
		result := resolver.ResolveTokenIDsForWindow(ctx, "BTC", "5m", candidate)
		if result.Status == marketresolver.StatusAvailable && result.UpTokenID != "" && result.DownTokenID != "" && result.ConditionID != "" {
			return result
		}
		attempts = append(attempts, fmt.Sprintf("%s status=%s source=%s", candidate.Format(time.RFC3339), result.Status, result.Source))
	}
	t.Fatalf("no live BTC 5m market resolved: %v", attempts)
	return marketresolver.ResolveResult{}
}

func clobMarketContainsToken(tokens []types.CLOBToken, tokenID string) bool {
	for _, token := range tokens {
		if token.TokenID == tokenID {
			return true
		}
	}
	return false
}

func assertUnitDecimal(t *testing.T, name, value string) {
	t.Helper()
	parseUnitDecimal(t, name, value)
}

func parseUnitDecimal(t *testing.T, name, value string) float64 {
	t.Helper()
	f := parseDecimal(t, name, value)
	if f < 0 || f > 1 {
		t.Fatalf("%s=%q outside [0,1]", name, value)
	}
	return f
}

func assertPositiveDecimal(t *testing.T, name, value string) {
	t.Helper()
	if f := parseDecimal(t, name, value); f <= 0 {
		t.Fatalf("%s=%q is not positive", name, value)
	}
}

func assertPositiveIfSet(t *testing.T, name string, value float64) {
	t.Helper()
	if value < 0 {
		t.Fatalf("%s=%f is negative", name, value)
	}
	if value != 0 && value <= 0 {
		t.Fatalf("%s=%f is not positive", name, value)
	}
}

func assertNonNegativeBps(t *testing.T, name string, value int) {
	t.Helper()
	if value < 0 || value > 10_000 {
		t.Fatalf("%s=%d outside [0,10000]", name, value)
	}
}

func parseDecimal(t *testing.T, name, value string) float64 {
	t.Helper()
	if value == "" {
		t.Fatalf("%s is empty", name)
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		t.Fatalf("%s=%q is not decimal: %v", name, value, err)
	}
	return f
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func shortToken(tokenID string) string {
	if len(tokenID) <= 12 {
		return tokenID
	}
	return tokenID[:12]
}
