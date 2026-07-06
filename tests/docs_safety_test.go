package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentationSafety(t *testing.T) {
	root := repositoryRoot(t)

	requiredDocs := []string{
		"docs/ARCHITECTURE.md",
		"docs/COMMANDS.md",
		"docs/COMPATIBILITY.md",
		"docs/COMPATIBILITY.json",
		"docs/SAFETY.md",
		"docs/MCP-OPENAPI.md",
		"docs/POLYGOLEM-OPEN-SOURCE-REINFORCEMENT-PLAN.md",
		"docs/POLYGOLEM-ROADMAP-MATRIX.md",
		"docs/POLYMARKET-COVERAGE-MATRIX.md",
		"docs/history/REFERENCE-RUST-CLI.md",
		"docs/adr/README.md",
		"docs/adr/0002-polymarket-api-interface-boundary.md",
		"docs/adr/0003-deposit-wallet-only-trading.md",
		"docs/adr/0004-public-sdk-boundary.md",
	}
	for _, requiredDoc := range requiredDocs {
		path := filepath.Join(root, filepath.FromSlash(requiredDoc))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected required documentation at %s: %v", requiredDoc, err)
		}
		if info.Size() == 0 {
			t.Fatalf("expected required documentation at %s to be non-empty", requiredDoc)
		}
	}

	readme := readRepositoryFile(t, root, "README.md")
	if !strings.Contains(readme, "polygolem") && !strings.Contains(readme, "Go Phase 1") {
		t.Fatalf("README.md must identify this repository as polygolem or Go Phase 1")
	}

	blockedPhrases := []string{
		"Rust CLI for Polymarket",
		"cargo install --path .",
		"brew install polymarket",
		"brew install polygolem",
		"polymarket setup",
		"polygolem setup",
		"polymarket wallet create",
		"polygolem wallet create",
	}
	activeUserDocs := []string{
		"README.md",
		"docs/ARCHITECTURE.md",
		"docs/COMMANDS.md",
		"docs/SAFETY.md",
	}
	for _, relativePath := range activeUserDocs {
		content := readRepositoryFile(t, root, relativePath)
		for _, blockedPhrase := range blockedPhrases {
			if strings.Contains(content, blockedPhrase) {
				t.Fatalf("%s contains unsupported Phase 1 phrase %q", relativePath, blockedPhrase)
			}
		}
	}

	loginDocs := []string{
		"README.md",
		"docs/ONBOARDING.md",
		"docs/BROWSER-SETUP.md",
		"docs/docs-site/src/content/docs/docs/guides/deposit-wallet-lifecycle.mdx",
		"docs/docs-site/src/content/docs/docs/guides/builder-auto.mdx",
	}
	for _, relativePath := range loginDocs {
		content := readRepositoryFile(t, root, relativePath)
		for _, required := range []string{
			"polygolem auth login",
			"Polymarket login signs with the EOA",
			"deposit wallet",
		} {
			if !strings.Contains(content, required) {
				t.Fatalf("%s must document headless auth login wording %q", relativePath, required)
			}
		}
		for _, blocked := range []string{
			"New users need one browser login",
			"browser login is required",
			"Requires browser login for new users",
			"pure headless onboarding is not possible",
			"Deposit-wallet-owned API keys cannot be created headlessly",
		} {
			if strings.Contains(content, blocked) {
				t.Fatalf("%s contains stale browser-first onboarding claim %q", relativePath, blocked)
			}
		}
	}

	reference := readRepositoryFile(t, root, "docs/history/REFERENCE-RUST-CLI.md")
	expectedReferenceText := "- `market-order`: builds, signs, and posts a market order through `post_order`."
	if !strings.Contains(reference, expectedReferenceText) {
		t.Fatalf("docs/REFERENCE-RUST-CLI.md must preserve exact upstream audit text %q", expectedReferenceText)
	}

	plan := readRepositoryFile(t, root, "docs/superpowers/plans/2026-05-06-polymarket-go-cli-phase-1.md")
	expectedPlanSnippet := `rg -n "live trading works|place live|create live order|market-order" docs README.md`
	if !strings.Contains(plan, expectedPlanSnippet) {
		t.Fatalf("phase 1 plan must preserve exact verification snippet %q", expectedPlanSnippet)
	}
	expectedPlanText := "Expected: no claim that live trading works in Phase 1."
	if !strings.Contains(plan, expectedPlanText) {
		t.Fatalf("phase 1 plan must preserve expected wording %q", expectedPlanText)
	}

	architecture := readRepositoryFile(t, root, "docs/ARCHITECTURE.md")
	for _, required := range []string{
		"Go SDK and CLI interface into Polymarket APIs and contracts",
		"Polygolem is not a bot or strategy engine",
		"Command handlers parse flags, call package APIs, and render output via\n`internal/output`",
		"Cobra command handlers must not contain protocol or trading business logic",
		"**Read-only** (default): public market data only",
		"**Paper**: local simulation",
		"**Live**: gated. Requires preflight + risk + funding gates to pass",
	} {
		if !strings.Contains(architecture, required) {
			t.Fatalf("docs/ARCHITECTURE.md must include architecture framing %q", required)
		}
	}

	coverage := readRepositoryFile(t, root, "docs/POLYMARKET-COVERAGE-MATRIX.md")
	for _, required := range []string{
		"RFQ | Typed RFQ request/quote/response models",
		"Public signer adapters",
		"Polymarket error normalization",
		"Reconciliation report",
		"docs/COMPATIBILITY.md",
		"generated compatibility contract",
		"upstream docs drift checker",
		"Agent/OpenAPI surfaces",
		"CTF split/merge/redeem helpers",
	} {
		if !strings.Contains(coverage, required) {
			t.Fatalf("docs/POLYMARKET-COVERAGE-MATRIX.md must include new reinforcement surface %q", required)
		}
	}

	planningDocs := []string{
		"docs/POLYGOLEM-OPEN-SOURCE-REINFORCEMENT-PLAN.md",
		"docs/POLYGOLEM-ROADMAP-MATRIX.md",
	}
	for _, relativePath := range planningDocs {
		content := readRepositoryFile(t, root, relativePath)
		if strings.Contains(content, "owner-scoped") {
			t.Fatalf("%s contains stale auth claim %q (should be EOA-bound)", relativePath, "owner-scoped")
		}
	}

	contextMD := readRepositoryFile(t, root, "CONTEXT.md")
	for _, required := range []string{
		"Polymarket API Interface",
		"EOA-Bound CLOB Auth",
		"POLY_1271 Order Signing",
		"ERC-7739 Wrapped Order Signature",
		"Safety-First Mutating Surface",
		"Read-Only by Default",
		"Capability Map",
		"Reconciliation Report",
	} {
		if !strings.Contains(contextMD, required) {
			t.Fatalf("CONTEXT.md must include core domain term %q", required)
		}
	}
	// Check that disallowed terms only appear in _Avoid_ lines, not as active definitions.
	for _, line := range strings.Split(contextMD, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "_Avoid_") {
			continue // _Avoid_ lines list exactly these phrases intentionally
		}
		if strings.Contains(trimmed, "deposit-wallet-owned API key") ||
			strings.Contains(trimmed, "deposit-wallet-owned L2 headers") {
			t.Fatalf("CONTEXT.md line %q contains disallowed active definition phrasing", trimmed)
		}
	}

	poly1271Docs := []string{
		"docs/POLY_1271-SIGNING.md",
		"docs/docs-site/src/content/docs/docs/concepts/poly-1271-signing.mdx",
		"docs/docs-site/src/content/docs/docs/guides/universal-client.mdx",
		"docs/docs-site/src/content/docs/docs/reference/clob-api.mdx",
	}
	for _, relativePath := range poly1271Docs {
		content := readRepositoryFile(t, root, relativePath)
		for _, blocked := range []string{
			"POLY_SIGNATURE = ERC-7739 wrapped ClobAuth",
			"L2 key must be bound to the **deposit wallet address**",
			"deposit-wallet-owned CLOB key",
			"deposit-wallet-owned L2 headers",
			"L1 + ERC-1271 owner",
		} {
			if strings.Contains(content, blocked) {
				t.Fatalf("%s contains stale POLY_1271 auth claim %q", relativePath, blocked)
			}
		}
	}

	commands := readRepositoryFile(t, root, "docs/COMMANDS.md")
	for _, required := range []string{
		"--json",
		"polygolem --json version",
		"set -euo pipefail",
		"jq",
	} {
		if !strings.Contains(commands, required) {
			t.Fatalf("docs/COMMANDS.md must include command automation guidance %q", required)
		}
	}

	docsIndex := readRepositoryFile(t, root, "docs/README.md")
	for _, required := range []string{
		"MCP-OPENAPI.md",
		"POLYGOLEM-OPEN-SOURCE-REINFORCEMENT-PLAN.md",
		"POLYGOLEM-ROADMAP-MATRIX.md",
		"POLYMARKET-COVERAGE-MATRIX.md",
		"adr/",
		"Architecture decision records",
	} {
		if !strings.Contains(docsIndex, required) {
			t.Fatalf("docs/README.md must index reinforcement doc %q", required)
		}
	}

	mcpOpenAPI := readRepositoryFile(t, root, "docs/MCP-OPENAPI.md")
	for _, required := range []string{
		"MCP and OpenAPI v1 are read-only only",
		"go run ./cmd/polygolem_mcp",
		"go run ./cmd/polygolem_openapi",
		"pkg/mcp.NewSDKReadOnlyHandlers",
	} {
		if !strings.Contains(mcpOpenAPI, required) {
			t.Fatalf("docs/MCP-OPENAPI.md must include read-only integration guidance %q", required)
		}
	}
	for _, blocked := range []string{
		"live order placement or cancellation;",
		"private-key signing;",
		"token approvals;",
		"bridge withdrawals;",
	} {
		if !strings.Contains(mcpOpenAPI, blocked) {
			t.Fatalf("docs/MCP-OPENAPI.md must document excluded mutating surface %q", blocked)
		}
	}

	safety := readRepositoryFile(t, root, "docs/SAFETY.md")
	for _, gate := range []string{
		"POLYMARKET_LIVE_PROFILE=on",
		"live_trading_enabled: true",
		"--confirm-live",
		"preflight",
	} {
		if !strings.Contains(safety, gate) {
			t.Fatalf("docs/SAFETY.md must document live gate %q", gate)
		}
	}
	for _, required := range []string{
		"Preflight checks config validity, wallet readiness, auth readiness, network consistency, API health, and chain consistency",
		"Automation must treat any preflight failure as terminal",
		"Dangerous operations include real order submission, payload signing, on-chain transactions, token approvals, private-key handling, and authenticated trading mutations",
		"Phase 1 intentionally contains no code path for those operations",
	} {
		if !strings.Contains(safety, required) {
			t.Fatalf("docs/SAFETY.md must include safety guidance %q", required)
		}
	}
}

func readRepositoryFile(t *testing.T, root, relativePath string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("expected to read %s: %v", relativePath, err)
	}
	return string(content)
}
