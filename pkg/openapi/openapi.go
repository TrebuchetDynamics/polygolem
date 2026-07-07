// Package openapi exposes a minimal OpenAPI description for safe read-only
// polygolem surfaces.
//
// The spec is intentionally generated in Go instead of introducing a schema
// framework dependency. It gives agents/proxies a stable discovery artifact
// while excluding live trading, signing, approvals, withdrawals, and other
// mutating operations.
package openapi

import (
	"sort"

	"github.com/TrebuchetDynamics/polygolem/pkg/capabilities"
)

// Spec returns a small OpenAPI 3.1 document for read-only local proxy/tooling
// experiments. Callers may marshal it directly as JSON.
func Spec() map[string]any {
	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "polygolem read-only API",
			"version":     "0.1.0",
			"description": "Read-only Polymarket discovery, data, orderbook, health, and diagnostics surfaces. Mutating trading and credentialed operations are deliberately excluded.",
		},
		"x-polygolem-capabilities": capabilities.ReadOnlyIDs(),
		"paths": map[string]any{
			"/health": get("Health check", "Check read-only Gamma and CLOB reachability."),
			"/diag":   get("Diagnostics", "Return redacted local diagnostics and endpoint configuration."),
			"/discover/search": getWithQuery("Search markets", "Search Polymarket Gamma markets.", map[string]any{
				"q":     stringParam("q", true, "Search query."),
				"limit": integerParam("limit", false, "Maximum number of rows."),
			}),
			"/data/positions": getWithQuery("Positions", "Read public Data API positions for a user address.", map[string]any{
				"user":  stringParam("user", true, "User or wallet address."),
				"limit": integerParam("limit", false, "Maximum number of rows."),
			}),
			"/orderbook/{token_id}": getWithPath("Order book", "Read a public CLOB order book by token id.", map[string]any{
				"token_id": pathStringParam("token_id", "CLOB token id."),
			}),
			"/marketdata/snapshot": getWithQuery("Market data snapshot", "Return a normalized market-data snapshot for a token.", map[string]any{
				"token_id": stringParam("token_id", true, "CLOB token id."),
			}),
		},
	}
}

func get(summary, description string) map[string]any {
	return map[string]any{"get": operation(summary, description, nil)}
}

func getWithQuery(summary, description string, params map[string]any) map[string]any {
	return map[string]any{"get": operation(summary, description, params)}
}

func getWithPath(summary, description string, params map[string]any) map[string]any {
	return map[string]any{"get": operation(summary, description, params)}
}

func operation(summary, description string, params map[string]any) map[string]any {
	op := map[string]any{
		"summary":     summary,
		"description": description,
		"responses": map[string]any{
			"200": map[string]any{"description": "Successful read-only response."},
		},
	}
	if len(params) > 0 {
		keys := make([]string, 0, len(params))
		for key := range params {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		ordered := make([]any, 0, len(params))
		for _, key := range keys {
			ordered = append(ordered, params[key])
		}
		op["parameters"] = ordered
	}
	return op
}

func stringParam(name string, required bool, description string) map[string]any {
	return parameter(name, "query", required, description, "string")
}

func pathStringParam(name string, description string) map[string]any {
	return parameter(name, "path", true, description, "string")
}

func integerParam(name string, required bool, description string) map[string]any {
	return parameter(name, "query", required, description, "integer")
}

func parameter(name, location string, required bool, description, typ string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          location,
		"required":    required,
		"description": description,
		"schema":      map[string]any{"type": typ},
	}
}
