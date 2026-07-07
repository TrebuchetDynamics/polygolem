package tests

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIRenameContractE2E(t *testing.T) {
	root := repositoryRoot(t)
	bin := filepath.Join(t.TempDir(), "polygolem")
	build := exec.Command("go", "build", "-o", bin, "./cmd/polygolem")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build polygolem: %v\n%s", err, out)
	}

	help := runCLI(t, root, bin, "--help")
	for _, want := range []string{"ping", "markets", "book", "exchange", "analytics", "wallet", "sim", "prices", "credentials", "risk", "doctor", "debug", "check-upstream", "tx", "builder-keys"} {
		if !strings.Contains(help, want) {
			t.Fatalf("root help missing renamed command %q\n%s", want, help)
		}
	}

	simHelp := runCLI(t, root, bin, "exchange", "simulate", "--help")
	if !strings.Contains(simHelp, "polygolem exchange simulate") || !strings.Contains(simHelp, "--limit-price") {
		t.Fatalf("simulate help does not expose renamed command and flags\n%s", simHelp)
	}

	for _, old := range []string{"health", "discover", "orderbook", "clob", "data", "intel", "deposit-wallet", "auth", "builder", "paper", "marketdata", "live", "preflight", "diag", "drift", "relayer"} {
		cmd := exec.Command(bin, old, "--help")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err == nil {
			t.Fatalf("legacy command %q still works\n%s", old, out)
		}
	}
}

func runCLI(t *testing.T, dir, bin string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", bin, strings.Join(args, " "), err, out)
	}
	return string(out)
}
