package marketresolver

import "strings"

// Outcome direction labels shared by every consumer that records crypto
// up/down market results.
const (
	OutcomeUp      = "up"
	OutcomeDown    = "down"
	OutcomeUnknown = "unknown"
)

// NormalizeOutcome canonicalizes a reported winning-outcome string to
// "up", "down", or "unknown".
func NormalizeOutcome(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case OutcomeUp:
		return OutcomeUp
	case OutcomeDown:
		return OutcomeDown
	default:
		return OutcomeUnknown
	}
}

// OutcomeForToken maps a winning token ID to "up"/"down"/"unknown" given the
// market's outcome-token pair. An empty winning token, or a token matching
// neither side, is "unknown".
func OutcomeForToken(winningTokenID, upTokenID, downTokenID string) string {
	winningTokenID = strings.TrimSpace(winningTokenID)
	if winningTokenID == "" {
		return OutcomeUnknown
	}
	switch winningTokenID {
	case strings.TrimSpace(upTokenID):
		return OutcomeUp
	case strings.TrimSpace(downTokenID):
		return OutcomeDown
	default:
		return OutcomeUnknown
	}
}

// OutcomeForToken maps a winning token ID against this market's token pair.
func (m CryptoMarket) OutcomeForToken(winningTokenID string) string {
	return OutcomeForToken(winningTokenID, m.UpTokenID, m.DownTokenID)
}
