package cli

import (
	"testing"
	"time"

	"github.com/divijg19/sage/internal/event"
)

func TestFilterChronicleEvents_QueryAndFilters(t *testing.T) {
	events := []event.Event{
		{
			Seq:       1,
			Timestamp: time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC),
			Project:   "alpha",
			Kind:      event.RecordKind,
			Title:     "Investigate auth cache",
			Content:   "Looked at session invalidation",
			Tags:      []string{"auth", "backend"},
		},
		{
			Seq:       2,
			Timestamp: time.Date(2026, 4, 21, 9, 0, 0, 0, time.UTC),
			Project:   "beta",
			Kind:      event.DecisionKind,
			Title:     "Choose sqlite wal",
			Content:   "Durability first",
			Tags:      []string{"storage"},
		},
	}

	filtered := filterChronicleEvents(events, chronicleFilters{
		Query:   "durability",
		Project: "beta",
		EnabledKinds: map[event.EntryKind]bool{
			event.DecisionKind: true,
		},
		EnabledTags: map[string]bool{
			"storage": true,
		},
	})

	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered event, got %d", len(filtered))
	}
	if filtered[0].Seq != 2 {
		t.Fatalf("expected seq 2, got %d", filtered[0].Seq)
	}
}

func TestBuildChronicleRows_CollapsesDaysAndExpandsEntries(t *testing.T) {
	events := []event.Event{
		{
			Seq:       1,
			Timestamp: time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC),
			Project:   "alpha",
			Kind:      event.RecordKind,
			Title:     "one",
			Content:   "first line\nsecond line",
		},
		{
			Seq:       2,
			Timestamp: time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC),
			Project:   "alpha",
			Kind:      event.DecisionKind,
			Title:     "two",
			Content:   "detail",
		},
		{
			Seq:       3,
			Timestamp: time.Date(2026, 4, 21, 11, 0, 0, 0, time.UTC),
			Project:   "beta",
			Kind:      event.RecordKind,
			Title:     "three",
			Content:   "detail",
		},
	}

	rows := buildChronicleRows(events, map[string]bool{
		"2026-04-20": true,
	}, map[int64]bool{
		3: true,
	})

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0].Kind != chronicleRowDay || rows[0].DayCount != 2 || rows[0].DayOpen {
		t.Fatalf("expected collapsed first day row, got %+v", rows[0])
	}
	if rows[1].Kind != chronicleRowDay || !rows[1].DayOpen || rows[1].DayCount != 1 {
		t.Fatalf("expected second day row, got %+v", rows[1])
	}
	if rows[2].Kind != chronicleRowEntry || !rows[2].EntryOpen || rows[2].Event.Seq != 3 {
		t.Fatalf("expected expanded third entry row, got %+v", rows[2])
	}
}
