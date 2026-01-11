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
	if all[0].Seq != 1 || all[1].Seq != 2 {
		t.Fatalf("expected seq 1,2 got %d,%d", all[0].Seq, all[1].Seq)
	}

	latest, err := s.Latest()
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

	if err := s.UpdateTagsBySeq(1, []string{"x", "y"}); err != nil {
		t.Fatalf("UpdateTagsBySeq: %v", err)
	}
	g, err := s.GetBySeq(1)
	if err != nil {
		t.Fatalf("GetBySeq: %v", err)
	}
	if g == nil {
		t.Fatalf("expected entry")
	}
	if len(g.Tags) != 2 || g.Tags[0] != "x" || g.Tags[1] != "y" {
		t.Fatalf("expected updated tags [x y], got %v", g.Tags)
	}
}

func TestStore_Latest_Empty(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sage.db")

	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	latest, err := s.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if latest != nil {
		t.Fatalf("expected nil latest on empty store")
	}
}
