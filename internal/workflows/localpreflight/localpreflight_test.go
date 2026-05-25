package localpreflight

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunnerReportsCoreLocalChecks(t *testing.T) {
	runner := New(Config{Version: "test-version"})

	result := runner.Run(context.Background())
	if !result.OK {
		t.Fatalf("preflight OK=false: %+v", result)
	}
	wantChecks := []string{"version", "output", "clob_builder_code"}
	if len(result.Checks) != len(wantChecks) {
		t.Fatalf("checks=%+v", result.Checks)
	}
	for i, want := range wantChecks {
		if result.Checks[i].Name != want || result.Checks[i].Status != "pass" {
			t.Fatalf("check[%d]=%+v want %s/pass", i, result.Checks[i], want)
		}
	}
}

func TestRunnerRejectsInvalidVersionAndBuilderCode(t *testing.T) {
	runner := New(Config{Version: "", BuilderCode: "0x1234"})

	result := runner.Run(context.Background())
	if result.OK {
		t.Fatal("preflight should fail")
	}
	failures := map[string]string{}
	for _, check := range result.Checks {
		if check.Status == "fail" {
			failures[check.Name] = check.Message
		}
	}
	if !strings.Contains(failures["version"], "version is empty") {
		t.Fatalf("version failure=%q", failures["version"])
	}
	if !strings.Contains(failures["clob_builder_code"], "builder code") {
		t.Fatalf("builder-code failure=%q", failures["clob_builder_code"])
	}
}

func TestWriteTextFormatsPassesAndFailures(t *testing.T) {
	runner := New(Config{Version: "", BuilderCode: "not-hex"})
	result := runner.Run(context.Background())

	var out bytes.Buffer
	if err := WriteText(&out, result); err != nil {
		t.Fatalf("WriteText returned error: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"preflight: failed\n",
		"- version: fail (version is empty)\n",
		"- output: pass\n",
		"- clob_builder_code: fail (builder code must be a 0x-prefixed bytes32 hex string)\n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("text missing %q:\n%s", want, text)
		}
	}
}
