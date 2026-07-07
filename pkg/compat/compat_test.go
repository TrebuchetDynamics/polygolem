package compat

import (
	"encoding/json"
	"testing"
)

func TestContractIncludesCapabilityAndErrorKinds(t *testing.T) {
	contract := Contract()
	if contract.Version == "" {
		t.Fatal("missing version")
	}
	clob := capabilityByID(t, contract, "clob.trading")
	if clob.Mode != "mutating" || clob.WalletMode != "deposit_wallet_only" {
		t.Fatalf("clob.trading=%+v", clob)
	}
	if !contains(clob.Auth, "private_key") {
		t.Fatalf("clob.trading auth=%v", clob.Auth)
	}
	userStream := capabilityByID(t, contract, "websocket.user")
	if userStream.Mode != "credentialed-read" {
		t.Fatalf("websocket.user mode=%q", userStream.Mode)
	}
	if !hasErrorKind(contract, "geoblocked") || !hasErrorKind(contract, "tick_size_mismatch") {
		t.Fatalf("missing expected error kinds: %+v", contract.ErrorKinds)
	}
}

func TestJSONIsStableAndParseable(t *testing.T) {
	body, err := JSON()
	if err != nil {
		t.Fatal(err)
	}
	var contract CompatibilityContract
	if err := json.Unmarshal(body, &contract); err != nil {
		t.Fatal(err)
	}
	if capabilityByID(t, contract, "clob.public_data").Mode != "read-only" {
		t.Fatalf("bad json contract: %s", body)
	}
	if body[len(body)-1] != '\n' {
		t.Fatal("json should end with newline for checked-in generated file")
	}
}

func capabilityByID(t *testing.T, contract CompatibilityContract, id string) Capability {
	t.Helper()
	for _, cap := range contract.Capabilities {
		if cap.ID == id {
			return cap
		}
	}
	t.Fatalf("missing capability %s", id)
	return Capability{}
}

func hasErrorKind(contract CompatibilityContract, kind string) bool {
	for _, row := range contract.ErrorKinds {
		if row.Kind == kind {
			return true
		}
	}
	return false
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
