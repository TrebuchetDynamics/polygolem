package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/output"
)

type jsonSchema struct {
	Title      string                     `json:"title"`
	Type       string                     `json:"type"`
	Required   []string                   `json:"required"`
	Properties map[string]json.RawMessage `json:"properties"`
}

func TestPublicRequestSchemasPinDTOFields(t *testing.T) {
	root := repositoryRoot(t)

	rfqSchema := readSchema(t, filepath.Join(root, "fixtures", "schemas", "rfq_request.schema.json"))
	if rfqSchema.Title != "Polygolem RFQ Request" {
		t.Fatalf("unexpected RFQ schema title %q", rfqSchema.Title)
	}
	for _, required := range []string{"side", "amount"} {
		if !contains(rfqSchema.Required, required) {
			t.Fatalf("RFQ schema missing required field %q", required)
		}
	}
	for _, property := range []string{"market_id", "token_id", "side", "amount", "expiration", "maker", "metadata"} {
		if _, ok := rfqSchema.Properties[property]; !ok {
			t.Fatalf("RFQ schema missing property %q", property)
		}
	}

	bridgeSchema := readSchema(t, filepath.Join(root, "fixtures", "schemas", "bridge_withdraw_request.schema.json"))
	if bridgeSchema.Title != "Polygolem Bridge Withdraw Request" {
		t.Fatalf("unexpected bridge schema title %q", bridgeSchema.Title)
	}
	for _, required := range []string{"fromChainId", "fromTokenAddress", "fromAmountBaseUnit", "toChainId", "toTokenAddress", "recipientAddress"} {
		if !contains(bridgeSchema.Required, required) {
			t.Fatalf("bridge withdraw schema missing required field %q", required)
		}
	}
	for _, property := range []string{"fromChainId", "fromTokenAddress", "fromAmountBaseUnit", "toChainId", "toTokenAddress", "recipientAddress", "quoteId"} {
		if _, ok := bridgeSchema.Properties[property]; !ok {
			t.Fatalf("bridge withdraw schema missing property %q", property)
		}
	}

	ctfSchema := readSchema(t, filepath.Join(root, "fixtures", "schemas", "ctf_operation_request.schema.json"))
	if ctfSchema.Title != "Polygolem CTF Operation Request" {
		t.Fatalf("unexpected CTF schema title %q", ctfSchema.Title)
	}
	for _, required := range []string{"operation", "collateralToken", "conditionId", "partition", "amountBaseUnits"} {
		if !contains(ctfSchema.Required, required) {
			t.Fatalf("CTF schema missing required field %q", required)
		}
	}
	for _, property := range []string{"operation", "collateralToken", "parentCollectionId", "conditionId", "partition", "amountBaseUnits"} {
		if _, ok := ctfSchema.Properties[property]; !ok {
			t.Fatalf("CTF schema missing property %q", property)
		}
	}
}

func TestCLIEnvelopeSchemaMatchesOutputContract(t *testing.T) {
	root := repositoryRoot(t)
	schemaPath := filepath.Join(root, "fixtures", "schemas", "cli_envelope.schema.json")
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read CLI envelope schema: %v", err)
	}
	var schema jsonSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("decode CLI envelope schema: %v", err)
	}
	if schema.Title != "Polygolem CLI JSON Envelope" || schema.Type != "object" {
		t.Fatalf("unexpected schema identity: %+v", schema)
	}
	for _, required := range []string{"ok", "version", "meta"} {
		if !contains(schema.Required, required) {
			t.Fatalf("schema missing required field %q", required)
		}
	}
	for _, property := range []string{"ok", "version", "data", "error", "meta"} {
		if _, ok := schema.Properties[property]; !ok {
			t.Fatalf("schema missing property %q", property)
		}
	}

	success := decodeEnvelopeFromWriter(t, func(buf *bytes.Buffer) error {
		return output.WriteSuccess(buf, "version", time.Unix(0, 0), map[string]string{"version": "test"})
	})
	assertEnvelopeConformsToSchemaSubset(t, success, true)

	failure := decodeEnvelopeFromWriter(t, func(buf *bytes.Buffer) error {
		return output.WriteErrorEnvelope(buf, "version", time.Unix(0, 0), output.Error{Code: "TEST", Message: "boom"})
	})
	assertEnvelopeConformsToSchemaSubset(t, failure, false)
}

func readSchema(t *testing.T, path string) jsonSchema {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema %s: %v", path, err)
	}
	var schema jsonSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("decode schema %s: %v", path, err)
	}
	if schema.Type != "object" {
		t.Fatalf("schema %s type=%q want object", path, schema.Type)
	}
	return schema
}

func decodeEnvelopeFromWriter(t *testing.T, write func(*bytes.Buffer) error) map[string]json.RawMessage {
	t.Helper()
	var buf bytes.Buffer
	if err := write(&buf); err != nil {
		t.Fatal(err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v\n%s", err, buf.String())
	}
	return envelope
}

func assertEnvelopeConformsToSchemaSubset(t *testing.T, envelope map[string]json.RawMessage, wantOK bool) {
	t.Helper()
	for _, required := range []string{"ok", "version", "meta"} {
		if _, ok := envelope[required]; !ok {
			t.Fatalf("envelope missing %q: %#v", required, envelope)
		}
	}
	var ok bool
	if err := json.Unmarshal(envelope["ok"], &ok); err != nil {
		t.Fatalf("ok is not boolean: %v", err)
	}
	if ok != wantOK {
		t.Fatalf("ok=%v want %v", ok, wantOK)
	}
	var version string
	if err := json.Unmarshal(envelope["version"], &version); err != nil || version != output.ContractVersion {
		t.Fatalf("version=%q err=%v", version, err)
	}
	var meta struct {
		Command    string `json:"command"`
		TS         string `json:"ts"`
		DurationMS int64  `json:"duration_ms"`
	}
	if err := json.Unmarshal(envelope["meta"], &meta); err != nil {
		t.Fatalf("meta shape: %v", err)
	}
	if meta.Command == "" || meta.TS == "" || meta.DurationMS < 0 {
		t.Fatalf("bad meta: %+v", meta)
	}
	if wantOK {
		if _, hasError := envelope["error"]; hasError {
			t.Fatalf("success envelope must not include error: %#v", envelope)
		}
		return
	}
	if _, hasError := envelope["error"]; !hasError {
		t.Fatalf("error envelope missing error: %#v", envelope)
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
