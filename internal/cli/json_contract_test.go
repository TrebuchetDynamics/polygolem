package cli

import (
	"errors"
	"testing"
)

func TestClassifyCommandErrorAddsUpstreamHTTPDetails(t *testing.T) {
	got := classifyCommandError(errors.New("clob create order: HTTP 400: maker address not allowed"))
	if got.Category != "protocol" || got.Code != "PROTOCOL_UPSTREAM_HTTP_ERROR" {
		t.Fatalf("classification=%s/%s, want protocol/PROTOCOL_UPSTREAM_HTTP_ERROR", got.Category, got.Code)
	}
	if got.Details["source"] != "clob" || got.Details["upstream_status"] != "400" {
		t.Fatalf("details=%v, want clob HTTP 400", got.Details)
	}
}

func TestClassifyCommandErrorSplitsNetworkAndChain(t *testing.T) {
	network := classifyCommandError(errors.New("health check: context deadline exceeded"))
	if network.Category != "network" || network.Details["source"] != "transport" {
		t.Fatalf("network classification=%+v", network)
	}

	chain := classifyCommandError(errors.New("submit transaction: execution reverted: insufficient funds"))
	if chain.Category != "chain" || chain.Details["source"] != "polygon_rpc" {
		t.Fatalf("chain classification=%+v", chain)
	}
}
