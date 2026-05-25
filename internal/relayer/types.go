package relayer

import (
	"encoding/json"
	"strconv"
	"strings"
)

// RelayerError is a structured error returned by the relayer API.
type RelayerError struct {
	Error string `json:"error"`
	Code  int    `json:"code,omitempty"`
}

func (r *RelayerError) UnmarshalJSON(data []byte) error {
	var raw struct {
		Error string          `json:"error"`
		Code  json.RawMessage `json:"code"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Error = raw.Error
	if len(raw.Code) == 0 || string(raw.Code) == "null" {
		return nil
	}
	var numeric int
	if err := json.Unmarshal(raw.Code, &numeric); err == nil {
		r.Code = numeric
		return nil
	}
	var text string
	if err := json.Unmarshal(raw.Code, &text); err == nil {
		if parsed, parseErr := strconv.Atoi(text); parseErr == nil {
			r.Code = parsed
		}
		return nil
	}
	return nil
}

// RelayerTransactionState maps the relayer's lifecycle states.
type RelayerTransactionState string

const (
	StateNew       RelayerTransactionState = "STATE_NEW"
	StateExecuted  RelayerTransactionState = "STATE_EXECUTED"
	StateMined     RelayerTransactionState = "STATE_MINED"
	StateInvalid   RelayerTransactionState = "STATE_INVALID"
	StateConfirmed RelayerTransactionState = "STATE_CONFIRMED"
	StateFailed    RelayerTransactionState = "STATE_FAILED"
)

func (s RelayerTransactionState) IsTerminal() bool {
	switch s {
	case StateMined, StateConfirmed, StateFailed, StateInvalid:
		return true
	}
	return false
}

func (s RelayerTransactionState) IsSuccess() bool {
	return s == StateMined || s == StateConfirmed
}

// DepositWalletCall is a single call within a WALLET batch.
type DepositWalletCall struct {
	Target string `json:"target"`
	Value  string `json:"value"`
	Data   string `json:"data"`
}

type WalletCreateRequest struct {
	Type string `json:"type"`
	From string `json:"from"`
	To   string `json:"to"`
}

// depositWalletParams is the nested params object within a WALLET batch.
type depositWalletParams struct {
	DepositWallet string              `json:"depositWallet"`
	Deadline      string              `json:"deadline"`
	Calls         []DepositWalletCall `json:"calls"`
}

// WalletBatchRequest is the payload for POST /submit with type=WALLET.
type WalletBatchRequest struct {
	Type                string              `json:"type"`
	From                string              `json:"from"`
	To                  string              `json:"to"`
	Nonce               string              `json:"nonce"`
	Signature           string              `json:"signature"`
	DepositWalletParams depositWalletParams `json:"depositWalletParams"`
}

// RelayerTransaction represents a submitted transaction tracked by the relayer.
type RelayerTransaction struct {
	TransactionID   string `json:"transactionID"`
	TransactionHash string `json:"transactionHash,omitempty"`
	From            string `json:"from"`
	To              string `json:"to"`
	ProxyAddress    string `json:"proxyAddress,omitempty"`
	Data            string `json:"data,omitempty"`
	Nonce           string `json:"nonce,omitempty"`
	Value           string `json:"value,omitempty"`
	State           string `json:"state"`
	Type            string `json:"type"`
	Metadata        string `json:"metadata,omitempty"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

func (r *RelayerTransaction) UnmarshalJSON(data []byte) error {
	var raw struct {
		TransactionIDCamel   json.RawMessage `json:"transactionID"`
		TransactionIDSnake   json.RawMessage `json:"transaction_id"`
		TransactionHashCamel json.RawMessage `json:"transactionHash"`
		TransactionHashSnake json.RawMessage `json:"transaction_hash"`
		From                 json.RawMessage `json:"from"`
		To                   json.RawMessage `json:"to"`
		ProxyAddressCamel    json.RawMessage `json:"proxyAddress"`
		ProxyAddressSnake    json.RawMessage `json:"proxy_address"`
		Data                 json.RawMessage `json:"data"`
		Nonce                json.RawMessage `json:"nonce"`
		Value                json.RawMessage `json:"value"`
		State                json.RawMessage `json:"state"`
		Type                 json.RawMessage `json:"type"`
		Metadata             json.RawMessage `json:"metadata"`
		CreatedAtCamel       json.RawMessage `json:"createdAt"`
		CreatedAtSnake       json.RawMessage `json:"created_at"`
		UpdatedAtCamel       json.RawMessage `json:"updatedAt"`
		UpdatedAtSnake       json.RawMessage `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.TransactionID = firstNonEmptyRawString(raw.TransactionIDCamel, raw.TransactionIDSnake)
	r.TransactionHash = firstNonEmptyRawString(raw.TransactionHashCamel, raw.TransactionHashSnake)
	r.From = rawStringOrNumber(raw.From)
	r.To = rawStringOrNumber(raw.To)
	r.ProxyAddress = firstNonEmptyRawString(raw.ProxyAddressCamel, raw.ProxyAddressSnake)
	r.Data = rawStringOrNumber(raw.Data)
	r.Nonce = rawStringOrNumber(raw.Nonce)
	r.Value = rawStringOrNumber(raw.Value)
	r.State = rawStringOrNumber(raw.State)
	r.Type = rawStringOrNumber(raw.Type)
	r.Metadata = rawStringOrNumber(raw.Metadata)
	r.CreatedAt = firstNonEmptyRawString(raw.CreatedAtCamel, raw.CreatedAtSnake)
	r.UpdatedAt = firstNonEmptyRawString(raw.UpdatedAtCamel, raw.UpdatedAtSnake)
	return nil
}

func firstNonEmptyRawString(values ...json.RawMessage) string {
	for _, value := range values {
		if decoded := rawStringOrNumber(value); decoded != "" {
			return decoded
		}
	}
	return ""
}

func rawStringOrNumber(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(string(raw))
	}
}

// NonceResponse is the response from GET /nonce?address=...&type=WALLET.
type NonceResponse struct {
	Nonce string `json:"nonce"`
}

func (r *NonceResponse) UnmarshalJSON(data []byte) error {
	var raw struct {
		Nonce json.RawMessage `json:"nonce"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Nonce = rawStringOrNumber(raw.Nonce)
	return nil
}

// DeployedResponse is the response from GET /deployed?address=...
type DeployedResponse struct {
	Deployed bool   `json:"deployed"`
	Address  string `json:"address,omitempty"`
}

func (r *DeployedResponse) UnmarshalJSON(data []byte) error {
	var raw struct {
		Deployed json.RawMessage `json:"deployed"`
		Address  json.RawMessage `json:"address"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Deployed = rawBoolOrFalse(raw.Deployed)
	r.Address = rawStringOrNumber(raw.Address)
	return nil
}

func rawBoolOrFalse(raw json.RawMessage) bool {
	switch strings.ToLower(rawStringOrNumber(raw)) {
	case "true", "1":
		return true
	default:
		return false
	}
}

// WalletCreateResponse is returned when WALLET-CREATE is accepted by the relayer.
// The relayer returns a transaction object with the created wallet address.
type WalletCreateResponse struct {
	TransactionID string `json:"transactionID"`
	WalletAddress string `json:"proxyAddress"`
	State         string `json:"state"`
}
