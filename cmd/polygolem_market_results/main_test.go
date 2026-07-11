package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/marketresults"
)

type fakeResolver struct {
	results map[string]*marketresults.Result
}

func (f fakeResolver) ResolveMarket(_ context.Context, market marketresults.MarketRef) (*marketresults.Result, error) {
	return f.results[market.ConditionID], nil
}

func TestRunEmitsSortedResolvedAndUnresolvedRows(t *testing.T) {
	at := time.Date(2026, 7, 10, 12, 5, 3, 0, time.UTC)
	resolver := fakeResolver{results: map[string]*marketresults.Result{
		"condition-b": {
			ConditionID:    "condition-b",
			WinningTokenID: "token-up",
			ResolvedAt:     at,
			ObservedAt:     at.Add(time.Second),
			Source:         "clob+gamma",
		},
	}}
	var out bytes.Buffer
	err := run(
		context.Background(),
		strings.NewReader(`{"markets":[{"condition_id":"condition-z","slug":"z","up_token_id":"z-up","down_token_id":"z-down"},{"condition_id":"condition-b","slug":"b","up_token_id":"b-up","down_token_id":"b-down"},{"condition_id":"condition-b","slug":"b","up_token_id":"b-up","down_token_id":"b-down"}]}`),
		&out,
		resolver,
	)
	if err != nil {
		t.Fatal(err)
	}
	var got response
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 1 || got.Results[0].ConditionID != "condition-b" {
		t.Fatalf("unexpected results: %#v", got.Results)
	}
	if len(got.Unresolved) != 1 || got.Unresolved[0] != "condition-z" {
		t.Fatalf("unexpected unresolved: %#v", got.Unresolved)
	}
}
