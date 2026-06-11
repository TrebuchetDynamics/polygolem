// Package httpsigner provides an optional HTTP-backed implementation of the
// public signers.Signer interface.
//
// It is intended for operators who isolate private keys behind a local signing
// service, KMS bridge, or custody adapter. The default polygolem binary does
// not require this package and still signs locally unless callers opt in.
package httpsigner

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/signers"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const defaultTimeout = 10 * time.Second

// Config controls the remote signer client. URL and BearerToken are required.
// Address and ChainID are returned through the signers.Signer identity methods
// and should describe the remote custody key.
type Config struct {
	URL         string
	BearerToken string
	Address     string
	ChainID     int64
	Timeout     time.Duration
	HTTPClient  *http.Client
}

// Signer calls a remote HTTP endpoint for each signing operation.
type Signer struct {
	config Config
	client *http.Client
}

var _ signers.Signer = (*Signer)(nil)

// New returns an HTTP-backed signer. The token is never included in returned
// errors; use signers.RedactSecret for any caller-side diagnostics.
func New(config Config) (*Signer, error) {
	if strings.TrimSpace(config.URL) == "" {
		return nil, fmt.Errorf("remote signer URL is required")
	}
	if strings.TrimSpace(config.BearerToken) == "" {
		return nil, fmt.Errorf("remote signer bearer token is required")
	}
	if config.Timeout <= 0 {
		config.Timeout = defaultTimeout
	}
	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &Signer{config: config, client: client}, nil
}

func (s *Signer) Address() string { return s.config.Address }
func (s *Signer) ChainID() int64  { return s.config.ChainID }

func (s *Signer) SignHash(hash [32]byte) ([]byte, error) {
	var payload = map[string]interface{}{
		"operation": "sign_hash",
		"address":   s.config.Address,
		"chain_id":  s.config.ChainID,
		"hash":      "0x" + hex.EncodeToString(hash[:]),
	}
	var resp signResponse
	if err := s.post(payload, &resp); err != nil {
		return nil, err
	}
	return decodeHexBytes(resp.Signature, 65, "signature")
}

func (s *Signer) SignTypedData(domainHash, structHash [32]byte) ([32]byte, error) {
	payload := map[string]interface{}{
		"operation":   "sign_typed_data_hashes",
		"address":     s.config.Address,
		"chain_id":    s.config.ChainID,
		"domain_hash": "0x" + hex.EncodeToString(domainHash[:]),
		"struct_hash": "0x" + hex.EncodeToString(structHash[:]),
	}
	var resp signResponse
	if err := s.post(payload, &resp); err != nil {
		return [32]byte{}, err
	}
	decoded, err := decodeHexBytes(firstNonEmpty(resp.Result, resp.Signature), 32, "typed-data result")
	if err != nil {
		return [32]byte{}, err
	}
	var out [32]byte
	copy(out[:], decoded)
	return out, nil
}

func (s *Signer) SignEIP712(typed apitypes.TypedData) ([]byte, error) {
	payload := map[string]interface{}{
		"operation":  "sign_eip712",
		"address":    s.config.Address,
		"chain_id":   s.config.ChainID,
		"typed_data": typed,
	}
	var resp signResponse
	if err := s.post(payload, &resp); err != nil {
		return nil, err
	}
	return decodeHexBytes(resp.Signature, 65, "signature")
}

type signResponse struct {
	Signature string `json:"signature"`
	Result    string `json:"result"`
}

func (s *Signer) post(payload interface{}, out *signResponse) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.URL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.config.BearerToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("remote signer request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("remote signer returned HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode remote signer response: %w", err)
	}
	return nil
}

func decodeHexBytes(value string, wantLen int, label string) ([]byte, error) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "0x")
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("invalid remote signer %s hex", label)
	}
	if len(decoded) != wantLen {
		return nil, fmt.Errorf("remote signer %s length=%d want %d", label, len(decoded), wantLen)
	}
	return decoded, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
