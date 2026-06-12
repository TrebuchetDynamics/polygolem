package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type conformanceArtifactHeader struct {
	SchemaVersion int    `json:"schema_version"`
	Family        string `json:"family"`
	Source        struct {
		ReferenceRepos []string `json:"reference_repos"`
		ReferencePaths []string `json:"reference_paths"`
		Notes          string   `json:"notes"`
	} `json:"source"`
}

func TestConformanceFixtureArtifactsExist(t *testing.T) {
	root := repositoryRoot(t)
	fixtures := map[string]string{
		"clob_auth_v2.json":      "clob-auth-v2",
		"order_v2_poly1271.json": "v2-poly1271-order-eip712",
		"builder_headers.json":   "builder-attribution-headers",
		"ctf_calldata.json":      "ctf-calldata",
	}
	for name, family := range fixtures {
		name, family := name, family
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, "fixtures", "conformance", name))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			var header conformanceArtifactHeader
			if err := json.Unmarshal(raw, &header); err != nil {
				t.Fatalf("decode fixture header: %v", err)
			}
			if header.SchemaVersion != 1 || header.Family != family {
				t.Fatalf("identity mismatch: %+v want family %q", header, family)
			}
			if len(header.Source.ReferenceRepos) == 0 || len(header.Source.ReferencePaths) == 0 || header.Source.Notes == "" {
				t.Fatalf("fixture must record reference repos, paths, and notes: %+v", header.Source)
			}
		})
	}
}

func TestConformanceFixtureReferenceCoverage(t *testing.T) {
	root := repositoryRoot(t)
	files := []string{
		"clob_auth_v2.json",
		"order_v2_poly1271.json",
		"builder_headers.json",
		"ctf_calldata.json",
	}
	seen := map[string]bool{}
	for _, name := range files {
		raw, err := os.ReadFile(filepath.Join(root, "fixtures", "conformance", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		var header conformanceArtifactHeader
		if err := json.Unmarshal(raw, &header); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		for _, repo := range header.Source.ReferenceRepos {
			seen[repo] = true
		}
	}
	for _, required := range []string{
		"opensource-projects/repos/ctf-exchange-v2",
		"opensource-projects/repos/Polymarket-golang",
		"opensource-projects/repos/go-builder-signing-sdk",
		"opensource-projects/repos/polymarket-kit/vendors/clob-client-v2",
	} {
		if !seen[required] {
			t.Fatalf("missing reference repo %s in conformance fixtures; seen=%v", required, seen)
		}
	}
}
