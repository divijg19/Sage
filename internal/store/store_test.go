package store

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/divijg19/sage/internal/event"
	_ "modernc.org/sqlite"
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

func TestStore_MigrateV1_AssignsDeterministicSeqAndIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sage.db")

	// Create a v1-style schema (no seq column).
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	createV1 := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT NOT NULL UNIQUE,
		timestamp TEXT NOT NULL,
		type TEXT NOT NULL,
		project TEXT NOT NULL,
		data TEXT NOT NULL
	);
	`
	if _, err := db.Exec(createV1); err != nil {
		t.Fatalf("create v1: %v", err)
	}

	base := time.Date(2026, 1, 9, 10, 0, 0, 0, time.UTC)
	e1 := event.Event{ID: "b", Timestamp: base.Add(2 * time.Minute), Project: "p", Kind: event.RecordKind, Title: "t2", Content: "c2"}
	e2 := event.Event{ID: "a", Timestamp: base.Add(1 * time.Minute), Project: "p", Kind: event.RecordKind, Title: "t1", Content: "c1"}

	// Insert out-of-order; migration must assign seq by timestamp ASC then id ASC.
	ins := `INSERT INTO events (id, timestamp, type, project, data) VALUES (?, ?, ?, ?, ?);`
	if _, err := db.Exec(ins, e1.ID, e1.Timestamp.Format(time.RFC3339), e1.Kind, e1.Project, mustJSON(t, e1)); err != nil {
		t.Fatalf("insert e1: %v", err)
	}
	if _, err := db.Exec(ins, e2.ID, e2.Timestamp.Format(time.RFC3339), e2.Kind, e2.Project, mustJSON(t, e2)); err != nil {
		t.Fatalf("insert e2: %v", err)
	}

	// Open via store (should migrate to v2).
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open (migrate): %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	if got[0].ID != "a" || got[0].Seq != 1 {
		t.Fatalf("expected first to be a/seq=1, got %q/%d", got[0].ID, got[0].Seq)
	}
	if got[1].ID != "b" || got[1].Seq != 2 {
		t.Fatalf("expected second to be b/seq=2, got %q/%d", got[1].ID, got[1].Seq)
	}

	// Idempotency: reopening should not change seq ordering.
	s2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open (second): %v", err)
	}
	got2, err := s2.List()
	if err != nil {
		t.Fatalf("List (second): %v", err)
	}
	if got2[0].ID != got[0].ID || got2[0].Seq != got[0].Seq || got2[1].ID != got[1].ID || got2[1].Seq != got[1].Seq {
		t.Fatalf("expected stable ordering across reopen")
	}
}

func mustJSON(t *testing.T, e event.Event) string {
	t.Helper()
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(b)
}
