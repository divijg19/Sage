package event

import "time"

// EntryKind represents the semantic kind of an entry.
type EntryKind string

const (
	RecordKind   EntryKind = "record"
	DecisionKind EntryKind = "decision"
	CommitKind   EntryKind = "commit"
)

// Event represents a single immutable cognitive entry.
type Event struct {
	Seq int64 `json:"-"`

	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Project   string    `json:"project"`

	Kind    EntryKind `json:"kind"`
	Title   string    `json:"title"`
	Content string    `json:"content"`

	Tags     []string          `json:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
