package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDiagJSONRedactsSecretsAndIncludesPreflight(t *testing.T) {
	secret := "0x1234567890abcdef1234567890abcdef"
	t.Setenv("POLYMARKET_PRIVATE_KEY", secret)
	t.Setenv("POLYMARKET_CLOB_URL", "https://example.invalid")

	stdout, stderr, err := executeRootForTest("--json", "diag")
	if err != nil {
		t.Fatalf("diag failed: %v\nstderr=%s", err, stderr)
	}
	if strings.Contains(stdout, secret) {
		t.Fatalf("diag leaked private key: %s", stdout)
	}
	got := parseJSONEnvelopeForTest(t, stdout)
	if !got.OK {
		t.Fatalf("ok=false: %s", stdout)
	}
	if got.Meta.Command != "diag" {
		t.Fatalf("command=%q want diag", got.Meta.Command)
	}
	var data struct {
		Version   string            `json:"version"`
		Endpoints map[string]string `json:"endpoints"`
		Env       map[string]struct {
			Set   bool   `json:"set"`
			Value string `json:"value"`
		} `json:"env"`
		Preflight struct {
			OK     bool                    `json:"ok"`
			Checks []struct{ Name string } `json:"checks"`
		} `json:"preflight"`
	}
	if err := json.Unmarshal(got.Data, &data); err != nil {
		t.Fatalf("decode diag data: %v\n%s", err, got.Data)
	}
	if data.Version != "test-version" {
		t.Fatalf("version=%q", data.Version)
	}
	if data.Endpoints["clob"] != "https://example.invalid" {
		t.Fatalf("clob endpoint=%q", data.Endpoints["clob"])
	}
	if !data.Env["POLYMARKET_PRIVATE_KEY"].Set || data.Env["POLYMARKET_PRIVATE_KEY"].Value == secret {
		t.Fatalf("private key env not redacted: %+v", data.Env["POLYMARKET_PRIVATE_KEY"])
	}
	if len(data.Preflight.Checks) == 0 {
		t.Fatal("expected preflight checks")
	}
}

func TestDiagTextIncludesEndpointsAndRedactedEnv(t *testing.T) {
	secret := "0xabcdefabcdefabcdef"
	t.Setenv("POLYMARKET_PRIVATE_KEY", secret)
	stdout, stderr, err := executeRootForTest("diag")
	if err != nil {
		t.Fatalf("diag failed: %v\nstderr=%s", err, stderr)
	}
	for _, want := range []string{"diag: polygolem test-version", "endpoints:", "env:", "POLYMARKET_PRIVATE_KEY"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("diag text missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, secret) {
		t.Fatalf("diag text leaked private key: %s", stdout)
	}
}
