package event

import "time"

type EventType string

const (
	LogEvent    EventType = "log"
	DecideEvent EventType = "decide"
	NoteEvent   EventType = "note"
)

type Event struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Type      EventType         `json:"type"`
	Project   string            `json:"project"`
	Concepts  []string          `json:"concepts,omitempty"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}
