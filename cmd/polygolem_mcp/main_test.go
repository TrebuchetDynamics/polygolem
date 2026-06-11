package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestPolygolemMCPStdioToolsList(t *testing.T) {
	cmd := exec.Command("go", "run", ".")
	cmd.Stdin = strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("polygolem_mcp failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "polygolem.health") {
		t.Fatalf("tools/list output missing health tool: %s", out)
	}
}
