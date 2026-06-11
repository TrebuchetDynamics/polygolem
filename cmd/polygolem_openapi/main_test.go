package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestPolygolemOpenAPICommandPrintsSpec(t *testing.T) {
	cmd := exec.Command("go", "run", ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("polygolem_openapi failed: %v\n%s", err, out)
	}
	text := string(out)
	for _, want := range []string{"\"openapi\": \"3.1.0\"", "/health", "/discover/search"} {
		if !strings.Contains(text, want) {
			t.Fatalf("OpenAPI output missing %q:\n%s", want, text)
		}
	}
}
