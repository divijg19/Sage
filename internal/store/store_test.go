package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/divijg19/sage/internal/event"
)

func TestStore_AppendListLatest(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sage.db")

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	base := time.Date(2026, 1, 9, 10, 0, 0, 0, time.UTC)

	e1 := event.Event{
		ID:        "1",
		Timestamp: base.Add(1 * time.Minute),
		Project:   "proj",
		Kind:      event.RecordKind,
		Title:     "t1",
		Content:   "c1",
		Tags:      []string{"a"},
	}
	e2 := event.Event{
		ID:        "2",
		Timestamp: base.Add(2 * time.Minute),
		Project:   "proj",
		Kind:      event.DecisionKind,
		Title:     "t2",
		Content:   "c2",
		Tags:      []string{"b"},
	}

	if err := s.Append(e1); err != nil {
		t.Fatalf("Append e1: %v", err)
	}
	if err := s.Append(e2); err != nil {
		t.Fatalf("Append e2: %v", err)
	}

	all, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 events, got %d", len(all))
	}

	latest, err := s.Latest("proj")
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if latest == nil {
		t.Fatalf("expected latest event")
	}
	if latest.ID != "2" {
		t.Fatalf("expected latest ID=2, got %q", latest.ID)
	}
	if latest.Title != "t2" {
		t.Fatalf("expected latest title t2, got %q", latest.Title)
	}
}

func TestStore_Latest_Empty(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sage.db")

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	latest, err := s.Latest("proj")
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if latest != nil {
		t.Fatalf("expected nil latest on empty store")
	}
}
