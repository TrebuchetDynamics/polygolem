package stream

import (
	"maps"
	"sync"
	"time"
)

// StreamStatsSnapshot is the JSON-friendly stream health snapshot emitted by
// CLI workflows and exposed by SDK clients.
type StreamStatsSnapshot struct {
	Type              string           `json:"type"`
	Stream            string           `json:"stream"`
	State             string           `json:"state"`
	AssetIDs          []string         `json:"asset_ids,omitempty"`
	Markets           []string         `json:"markets,omitempty"`
	MessagesReceived  int64            `json:"messages_received"`
	DuplicateMessages int64            `json:"duplicate_messages"`
	InvalidMessages   int64            `json:"invalid_messages"`
	Reconnects        int64            `json:"reconnects"`
	ConnectedAt       string           `json:"connected_at,omitempty"`
	DisconnectedAt    string           `json:"disconnected_at,omitempty"`
	LastReconnectAt   string           `json:"last_reconnect_at,omitempty"`
	LastMessageAt     string           `json:"last_message_at,omitempty"`
	EventCounts       map[string]int64 `json:"event_counts,omitempty"`
}

// StreamStats records lifecycle and message counters for a WebSocket stream.
type StreamStats struct {
	mu sync.Mutex

	stream      string
	state       string
	assetIDs    []string
	markets     []string
	messages    int64
	duplicates  int64
	invalid     int64
	reconnects  int64
	eventCounts map[string]int64

	connectedAt     time.Time
	disconnectedAt  time.Time
	lastReconnectAt time.Time
	lastMessageAt   time.Time
}

// NewStreamStats creates stats for a stream kind such as market or user.
func NewStreamStats(stream string) *StreamStats {
	return &StreamStats{stream: stream, state: "idle", eventCounts: make(map[string]int64)}
}

func (s *StreamStats) SetSubscriptions(assetIDs, markets []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.assetIDs = append([]string(nil), assetIDs...)
	s.markets = append([]string(nil), markets...)
}

func (s *StreamStats) MarkConnected(at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = "connected"
	s.connectedAt = at.UTC()
	s.disconnectedAt = time.Time{}
}

func (s *StreamStats) MarkDisconnected(at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = "disconnected"
	s.disconnectedAt = at.UTC()
}

func (s *StreamStats) RecordMessage(at time.Time) {
	s.RecordEvent("", at)
}

func (s *StreamStats) RecordEvent(eventType string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages++
	if eventType != "" {
		s.eventCounts[eventType]++
	}
	s.lastMessageAt = at.UTC()
}

func (s *StreamStats) RecordDuplicate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.duplicates++
}

func (s *StreamStats) RecordInvalid() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invalid++
}

func (s *StreamStats) RecordReconnect(at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reconnects++
	s.lastReconnectAt = at.UTC()
}

func (s *StreamStats) Snapshot() StreamStatsSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return StreamStatsSnapshot{
		Type:              "stream_stats",
		Stream:            s.stream,
		State:             s.state,
		AssetIDs:          append([]string(nil), s.assetIDs...),
		Markets:           append([]string(nil), s.markets...),
		MessagesReceived:  s.messages,
		DuplicateMessages: s.duplicates,
		InvalidMessages:   s.invalid,
		Reconnects:        s.reconnects,
		ConnectedAt:       formatStatsTime(s.connectedAt),
		DisconnectedAt:    formatStatsTime(s.disconnectedAt),
		LastReconnectAt:   formatStatsTime(s.lastReconnectAt),
		LastMessageAt:     formatStatsTime(s.lastMessageAt),
		EventCounts:       maps.Clone(s.eventCounts),
	}
}

func formatStatsTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
