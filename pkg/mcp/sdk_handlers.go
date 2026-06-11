package mcp

import (
	"context"
	"fmt"
	"strconv"

	sdkclob "github.com/TrebuchetDynamics/polygolem/pkg/clob"
	sdkdata "github.com/TrebuchetDynamics/polygolem/pkg/data"
	"github.com/TrebuchetDynamics/polygolem/pkg/gamma"
	"github.com/TrebuchetDynamics/polygolem/pkg/marketdata"
	"github.com/TrebuchetDynamics/polygolem/pkg/types"
)

// NewSDKReadOnlyHandlers wires concrete polygolem SDK clients into the safe MCP
// read-only handler set. Nil clients are skipped so deployments can expose only
// the surfaces they have configured.
func NewSDKReadOnlyHandlers(config HandlerConfig, gammaClient *gamma.Client, dataClient *sdkdata.Client, clobClient *sdkclob.Client) map[string]ToolHandler {
	adapters := ReadOnlyAdapters{}
	if gammaClient != nil || clobClient != nil {
		adapters.Health = func(ctx context.Context) (any, error) {
			out := map[string]string{}
			if gammaClient != nil {
				if _, err := gammaClient.HealthCheck(ctx); err != nil {
					out["gamma"] = "error: " + err.Error()
				} else {
					out["gamma"] = "ok"
				}
			}
			if clobClient != nil {
				if err := clobClient.Health(ctx); err != nil {
					out["clob"] = "error: " + err.Error()
				} else {
					out["clob"] = "ok"
				}
			}
			return out, nil
		}
	}
	if gammaClient != nil {
		adapters.DiscoverSearch = func(ctx context.Context, args map[string]any) (any, error) {
			limit := intArg(args, "limit")
			params := &types.SearchParams{Q: stringArg(args, "query")}
			if limit > 0 {
				params.LimitPerType = &limit
			}
			return gammaClient.Search(ctx, params)
		}
	}
	if dataClient != nil {
		adapters.DataPositions = func(ctx context.Context, args map[string]any) (any, error) {
			return dataClient.CurrentPositionsWithLimit(ctx, stringArg(args, "user"), intArg(args, "limit"))
		}
	}
	if clobClient != nil {
		adapters.OrderBook = func(ctx context.Context, args map[string]any) (any, error) {
			return clobClient.OrderBook(ctx, stringArg(args, "token_id"))
		}
	}
	config.Adapters = mergeReadOnlyAdapters(config.Adapters, adapters)
	return NewReadOnlyHandlers(config)
}

// NewMarketDataSnapshotHandler returns an MCP handler backed by an in-memory
// marketdata.Tracker. It is read-only: callers feed the tracker from their own
// websocket flow and MCP only reads the latest snapshot.
func NewMarketDataSnapshotHandler(tracker *marketdata.Tracker) ToolHandler {
	if tracker == nil {
		return nil
	}
	return func(ctx context.Context, args map[string]any) (any, error) {
		assetID := stringArg(args, "token_id")
		if assetID == "" {
			assetID = stringArg(args, "asset_id")
		}
		if assetID == "" {
			return nil, fmt.Errorf("token_id is required")
		}
		snapshot, ok := tracker.Snapshot(assetID)
		if !ok {
			return nil, fmt.Errorf("marketdata snapshot for %s is not available", assetID)
		}
		return snapshot, nil
	}
}

func mergeReadOnlyAdapters(base, add ReadOnlyAdapters) ReadOnlyAdapters {
	if base.Health == nil {
		base.Health = add.Health
	}
	if base.DiscoverSearch == nil {
		base.DiscoverSearch = add.DiscoverSearch
	}
	if base.DataPositions == nil {
		base.DataPositions = add.DataPositions
	}
	if base.OrderBook == nil {
		base.OrderBook = add.OrderBook
	}
	if base.MarketDataSnapshot == nil {
		base.MarketDataSnapshot = add.MarketDataSnapshot
	}
	return base
}

func stringArg(args map[string]any, name string) string {
	if value, ok := args[name].(string); ok {
		return value
	}
	return ""
}

func intArg(args map[string]any, name string) int {
	value, ok := args[name]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		parsed, err := strconv.Atoi(v)
		if err == nil {
			return parsed
		}
	}
	return 0
}
