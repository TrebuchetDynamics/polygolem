package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestRepositoryHygiene(t *testing.T) {
	root := repositoryRoot(t)

	ciPath := filepath.Join(root, ".github", "workflows", "ci.yml")
	ci, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("expected Go CI workflow at %s: %v", ciPath, err)
	}

	ciContent := string(ci)
	for _, required := range []string{
		"actions/setup-go@",
		"go-version-file: go.mod",
		"git ls-files -z '*.go' ':!:opensource-projects/**'",
		"xargs -0 gofmt -w",
		"go vet ./...",
		"go test -short ./...",
		"git diff --exit-code",
		"actions/setup-node@",
		"npm --prefix docs/docs-site ci",
		"npm --prefix docs/docs-site run build",
	} {
		if !strings.Contains(ciContent, required) {
			t.Fatalf("expected CI workflow to contain %q", required)
		}
	}

	for _, unsafePath := range []string{
		"Cargo.toml",
		"Cargo.lock",
		"Formula",
		"install.sh",
		"scripts/install.sh",
		"scripts/release.sh",
		"src",
		".github/workflows/release.yml",
		"cmd/polymarket",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(unsafePath))); err == nil {
			t.Fatalf("unsafe Rust/prototype path still exists: %s", unsafePath)
		} else if !os.IsNotExist(err) {
			t.Fatalf("could not inspect %s: %v", unsafePath, err)
		}
	}
	if entries, err := os.ReadDir(filepath.Join(root, "scripts")); err == nil {
		for _, entry := range entries {
			if entry.Name() != "playwright-capture" && entry.Name() != "coverage.sh" && entry.Name() != "live-smoke.sh" {
				t.Fatalf("unexpected scripts path still exists: scripts/%s", entry.Name())
			}
		}
	} else if !os.IsNotExist(err) {
		t.Fatalf("could not inspect scripts: %v", err)
	}

	// pkg/ is the approved public SDK boundary.
	if _, err := os.Stat(filepath.Join(root, "pkg/clob")); err != nil {
		t.Fatalf("pkg/clob public boundary is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "pkg/contracts")); err != nil {
		t.Fatalf("pkg/contracts public boundary is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "pkg/settlement")); err != nil {
		t.Fatalf("pkg/settlement public boundary is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "pkg/orderbook")); err != nil {
		t.Fatalf("pkg/orderbook public boundary is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "pkg/stream")); err != nil {
		t.Fatalf("pkg/stream public boundary is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "pkg/signers")); err != nil {
		t.Fatalf("pkg/signers public boundary is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "pkg/rfq")); err != nil {
		t.Fatalf("pkg/rfq public boundary is missing: %v", err)
	}
}

func TestPublicPackageInventoryIsDocumented(t *testing.T) {
	root := repositoryRoot(t)
	readme := string(readFile(t, filepath.Join(root, "README.md")))
	architecture := string(readFile(t, filepath.Join(root, "docs", "ARCHITECTURE.md")))
	sdkReference := string(readFile(t, filepath.Join(root, "docs", "docs-site", "src", "content", "docs", "docs", "reference", "sdk.mdx")))

	for _, pkg := range publicPackageDirs(t, root) {
		readmeLink := "[`" + pkg + "`](" + pkg + ")"
		if !strings.Contains(readme, readmeLink) {
			t.Fatalf("README.md missing public SDK package %s", pkg)
		}
		architectureRow := "| `" + pkg + "` |"
		if !strings.Contains(architecture, architectureRow) {
			t.Fatalf("docs/ARCHITECTURE.md missing public SDK package %s", pkg)
		}
		sdkHeading := "## `" + pkg + "`"
		if !strings.Contains(sdkReference, sdkHeading) {
			t.Fatalf("docs-site SDK reference missing public SDK package %s", pkg)
		}
	}
}

func TestPublicSDKFixtureImportsEveryPublicPackage(t *testing.T) {
	root := repositoryRoot(t)
	fixture := string(readFile(t, filepath.Join(root, "tests", "public_sdk_boundary_test.go")))

	for _, pkg := range publicPackageDirs(t, root) {
		importPath := "github.com/TrebuchetDynamics/polygolem/" + pkg
		if !strings.Contains(fixture, importPath) {
			t.Fatalf("public SDK compile fixture missing import for %s", pkg)
		}
	}
}

func TestRepositoryDoesNotPublishResolvedRemoteBlocker(t *testing.T) {
	root := repositoryRoot(t)

	todo, err := os.ReadFile(filepath.Join(root, "TODO.md"))
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("could not inspect TODO.md: %v", err)
	}

	content := string(todo)
	for _, stale := range []string{
		"TrebuchetDynamics/polygolem.git",
		"[BLOCKED] Push phase-1-tdd to origin",
	} {
		if strings.Contains(content, stale) {
			t.Fatalf("TODO.md contains resolved remote blocker %q", stale)
		}
	}
}

func publicPackageDirs(t *testing.T, root string) []string {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(root, "pkg"))
	if err != nil {
		t.Fatalf("read pkg dir: %v", err)
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "experimental" {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(root, "pkg", entry.Name(), "*.go"))
		if err != nil {
			t.Fatalf("inspect pkg/%s: %v", entry.Name(), err)
		}
		if len(matches) > 0 {
			out = append(out, "pkg/"+entry.Name())
		}
	}
	sort.Strings(out)
	return out
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve test file path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(filename), ".."))
}
