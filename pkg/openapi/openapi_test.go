package openapi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSpecKeepsPathAndQueryParametersDistinct(t *testing.T) {
	spec := Spec()
	paths := spec["paths"].(map[string]any)

	orderBookParam := firstParameterForPath(t, paths, "/orderbook/{token_id}")
	if orderBookParam["name"] != "token_id" || orderBookParam["in"] != "path" || orderBookParam["required"] != true {
		t.Fatalf("orderbook token_id must be required path param: %#v", orderBookParam)
	}

	marketDataParam := firstParameterForPath(t, paths, "/marketdata/snapshot")
	if marketDataParam["name"] != "token_id" || marketDataParam["in"] != "query" || marketDataParam["required"] != true {
		t.Fatalf("marketdata token_id must be required query param: %#v", marketDataParam)
	}
}

func TestSpecIncludesCapabilityMetadata(t *testing.T) {
	spec := Spec()
	caps, ok := spec["x-polygolem-capabilities"].([]string)
	if !ok || len(caps) == 0 {
		t.Fatalf("missing capability metadata: %#v", spec["x-polygolem-capabilities"])
	}
	for _, blocked := range []string{"clob.trading", "bridge.funding", "relayer.deposit_wallet"} {
		for _, cap := range caps {
			if cap == blocked {
				t.Fatalf("read-only OpenAPI capability metadata includes mutating %s", blocked)
			}
		}
	}
	for _, required := range []string{"gamma.markets", "data.positions", "clob.public_data"} {
		if !containsString(caps, required) {
			t.Fatalf("missing read-only capability metadata %s in %v", required, caps)
		}
	}
}

func TestSpecExposesOnlyReadOnlyPaths(t *testing.T) {
	spec := Spec()
	if spec["openapi"] != "3.1.0" {
		t.Fatalf("openapi=%v", spec["openapi"])
	}
	paths, ok := spec["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		t.Fatalf("missing paths: %#v", spec["paths"])
	}
	for _, required := range []string{"/health", "/diag", "/discover/search", "/data/positions", "/orderbook/{token_id}", "/marketdata/snapshot"} {
		if _, ok := paths[required]; !ok {
			t.Fatalf("spec missing read-only path %s", required)
		}
	}
	encoded, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(string(encoded))
	for _, blocked := range []string{"create-order", "withdraw", "approve", "signing", "trade"} {
		if strings.Contains(lower, blocked) {
			t.Fatalf("spec contains blocked mutating wording %q: %s", blocked, encoded)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func firstParameterForPath(t *testing.T, paths map[string]any, path string) map[string]any {
	t.Helper()
	pathItem, ok := paths[path].(map[string]any)
	if !ok {
		t.Fatalf("missing path %s", path)
	}
	get, ok := pathItem["get"].(map[string]any)
	if !ok {
		t.Fatalf("missing GET for %s", path)
	}
	params, ok := get["parameters"].([]any)
	if !ok || len(params) == 0 {
		t.Fatalf("missing parameters for %s: %#v", path, get["parameters"])
	}
	param, ok := params[0].(map[string]any)
	if !ok {
		t.Fatalf("bad first parameter for %s: %#v", path, params[0])
	}
	return param
}
