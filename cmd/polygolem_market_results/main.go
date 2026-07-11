// polygolem_market_results is a read-only batch adapter for non-Go consumers.
// It reads one JSON request from stdin and writes one deterministic JSON result.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/marketresults"
)

type resolver interface {
	ResolveMarket(context.Context, marketresults.MarketRef) (*marketresults.Result, error)
}

type request struct {
	Markets []marketresults.MarketRef `json:"markets"`
}

type response struct {
	SchemaVersion uint32                 `json:"schema_version"`
	Results       []marketresults.Result `json:"results"`
	Unresolved    []string               `json:"unresolved"`
}

func run(ctx context.Context, in io.Reader, out io.Writer, source resolver) error {
	decoder := json.NewDecoder(io.LimitReader(in, 1<<20))
	decoder.DisallowUnknownFields()
	var req request
	if err := decoder.Decode(&req); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("request must contain exactly one JSON value")
	}
	markets := make([]marketresults.MarketRef, 0, len(req.Markets))
	seen := make(map[string]struct{}, len(req.Markets))
	for _, market := range req.Markets {
		market.ConditionID = strings.TrimSpace(market.ConditionID)
		market.Slug = strings.TrimSpace(market.Slug)
		market.UpTokenID = strings.TrimSpace(market.UpTokenID)
		market.DownTokenID = strings.TrimSpace(market.DownTokenID)
		if market.ConditionID == "" || market.Slug == "" || market.UpTokenID == "" || market.DownTokenID == "" {
			return fmt.Errorf("market reference contains an empty required value")
		}
		if _, exists := seen[market.ConditionID]; exists {
			continue
		}
		seen[market.ConditionID] = struct{}{}
		markets = append(markets, market)
	}
	if len(markets) > 256 {
		return fmt.Errorf("markets exceeds maximum batch size 256")
	}
	sort.Slice(markets, func(i, j int) bool { return markets[i].ConditionID < markets[j].ConditionID })
	result := response{SchemaVersion: 1}
	for _, market := range markets {
		row, err := source.ResolveMarket(ctx, market)
		if err != nil {
			return fmt.Errorf("resolve %s: %w", market.ConditionID, err)
		}
		if row == nil {
			result.Unresolved = append(result.Unresolved, market.ConditionID)
			continue
		}
		result.Results = append(result.Results, *row)
	}
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(result)
}

func main() {
	clobURL := flag.String("clob-url", "", "override the public CLOB base URL")
	gammaURL := flag.String("gamma-url", "", "override the public Gamma base URL")
	timeout := flag.Duration("timeout", 20*time.Second, "batch timeout")
	flag.Parse()
	if flag.NArg() != 0 || *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "invalid arguments")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := run(ctx, os.Stdin, os.Stdout, marketresults.NewResolver(*clobURL, *gammaURL)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
