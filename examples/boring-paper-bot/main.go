package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/marketresolver"
	"github.com/TrebuchetDynamics/polygolem/pkg/universal"
)

type paperDecision struct {
	TS          string  `json:"ts"`
	Asset       string  `json:"asset"`
	Interval    string  `json:"interval"`
	ConditionID string  `json:"condition_id"`
	TokenID     string  `json:"token_id"`
	Side        string  `json:"side"`
	Price       float64 `json:"price"`
	Size        float64 `json:"size"`
	PaperCost   float64 `json:"paper_cost"`
	Reason      string  `json:"reason"`
}

func main() {
	asset := env("POLYGOLEM_BORING_ASSET", "BTC")
	interval := env("POLYGOLEM_BORING_INTERVAL", "5m")
	size := envFloat("POLYGOLEM_BORING_SIZE", 1)
	maxPrice := envFloat("POLYGOLEM_BORING_MAX_PRICE", 0.55)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resolver := marketresolver.NewResolver("")
	resolved := resolver.ResolveTokenIDs(ctx, asset, interval)
	if resolved.Status != marketresolver.StatusAvailable {
		log.Fatalf("market unresolved: status=%s source=%s", resolved.Status, resolved.Source)
	}

	client := universal.NewClient(universal.Config{})
	upAsk, err := client.Price(ctx, resolved.UpTokenID, "buy")
	if err != nil {
		log.Fatalf("up price: %v", err)
	}
	downAsk, err := client.Price(ctx, resolved.DownTokenID, "buy")
	if err != nil {
		log.Fatalf("down price: %v", err)
	}

	up := mustPrice("up", upAsk)
	down := mustPrice("down", downAsk)
	token, side, price := resolved.UpTokenID, "up", up
	if down < up {
		token, side, price = resolved.DownTokenID, "down", down
	}
	if price > maxPrice {
		log.Fatalf("no paper trade: best %s ask %.4f exceeds max %.4f", side, price, maxPrice)
	}

	decision := paperDecision{
		TS:          time.Now().UTC().Format(time.RFC3339),
		Asset:       asset,
		Interval:    interval,
		ConditionID: resolved.ConditionID,
		TokenID:     token,
		Side:        side,
		Price:       price,
		Size:        size,
		PaperCost:   price * size,
		Reason:      "cheaper side below max price; paper log only, no order submitted",
	}
	if err := json.NewEncoder(os.Stdout).Encode(decision); err != nil {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || f <= 0 {
			log.Fatalf("%s must be positive decimal", key)
		}
		return f
	}
	return fallback
}

func mustPrice(label, value string) float64 {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil || f <= 0 || f > 1 {
		log.Fatalf("%s price %q invalid: %v", label, value, err)
	}
	return f
}

func init() {
	fmt.Fprintln(os.Stderr, "boring-paper-bot: read-only discovery + paper JSONL decision; no signing/submission")
}
