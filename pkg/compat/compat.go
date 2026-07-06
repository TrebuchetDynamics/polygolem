// Package compat exposes the machine-readable Polygolem compatibility
// contract for downstream services and UIs.
package compat

import (
	"encoding/json"

	"github.com/TrebuchetDynamics/polygolem/pkg/capabilities"
	"github.com/TrebuchetDynamics/polygolem/pkg/polyerrors"
)

const Version = "v1"

type CompatibilityContract struct {
	Version      string       `json:"version"`
	Capabilities []Capability `json:"capabilities"`
	ErrorKinds   []ErrorKind  `json:"error_kinds"`
}

type Capability struct {
	ID          string   `json:"id"`
	Service     string   `json:"service"`
	Mode        string   `json:"mode"`
	Auth        []string `json:"auth"`
	WalletMode  string   `json:"wallet_mode"`
	SDKPackages []string `json:"sdk_packages"`
	CLI         []string `json:"cli"`
	Summary     string   `json:"summary"`
}

type ErrorKind struct {
	Kind    string `json:"kind"`
	Meaning string `json:"meaning"`
}

func Contract() CompatibilityContract {
	return CompatibilityContract{
		Version:      Version,
		Capabilities: capabilityRows(),
		ErrorKinds:   errorRows(),
	}
}

func JSON() ([]byte, error) {
	body, err := json.MarshalIndent(Contract(), "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

func capabilityRows() []Capability {
	caps := capabilities.All()
	out := make([]Capability, 0, len(caps))
	for _, cap := range caps {
		out = append(out, Capability{
			ID:          cap.ID,
			Service:     cap.Service,
			Mode:        Mode(cap),
			Auth:        authStrings(cap.Auth),
			WalletMode:  string(cap.WalletMode),
			SDKPackages: append([]string(nil), cap.SDKPackages...),
			CLI:         append([]string(nil), cap.CLI...),
			Summary:     cap.Summary,
		})
	}
	return out
}

func Mode(cap capabilities.Capability) string {
	if cap.Mutating {
		return "mutating"
	}
	if cap.ReadOnly {
		return "read-only"
	}
	return "credentialed-read"
}

func authStrings(auth []capabilities.AuthRequirement) []string {
	out := make([]string, 0, len(auth))
	for _, item := range auth {
		out = append(out, string(item))
	}
	return out
}

func errorRows() []ErrorKind {
	kinds := polyerrors.Kinds()
	out := make([]ErrorKind, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, ErrorKind{Kind: string(kind), Meaning: ErrorKindMeaning(kind)})
	}
	return out
}

func ErrorKindMeaning(kind polyerrors.Kind) string {
	switch kind {
	case polyerrors.RateLimited:
		return "Upstream rate limit or too-many-requests response."
	case polyerrors.Geoblocked:
		return "Upstream geoblock or restricted-jurisdiction response."
	case polyerrors.AuthRejected:
		return "Credential, API key, signature, or authorization rejection."
	case polyerrors.TickSizeMismatch:
		return "Order price does not align with the market tick size."
	case polyerrors.MarketClosed:
		return "Market is closed, resolved, or unavailable for new action."
	case polyerrors.InsufficientFunds:
		return "EOA or deposit wallet balance is insufficient for the requested action."
	case polyerrors.UpstreamUnavailable:
		return "Upstream Polymarket service returned a 5xx/outage-style response."
	default:
		return "Unclassified upstream error after redaction."
	}
}
