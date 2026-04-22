package cli

import (
	"reflect"
	"strings"
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

func TestChronicleInlinePreview_TrimsAndCapsLines(t *testing.T) {
	body := "\n  one\n two\nthree\nfour\nfive\n"
	got := chronicleInlinePreview(body)
	if got != "one\n two\nthree\nfour" {
		t.Fatalf("unexpected inline preview: %q", got)
	}
}

func TestChronicleUnionTags_CollectsDedupedNormalizedTags(t *testing.T) {
	events := []event.Event{
		{Tags: []string{"Ops", "backend"}},
		{Tags: []string{" qa ", "ops"}},
	}

	got := chronicleUnionTags([]string{"Auth, backend", "auth"}, events)
	want := []string{"auth", "backend", "ops", "qa"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected tags\nwant: %#v\n got: %#v", want, got)
	}
}

func TestChronicleProjectOptions_SortsAndMapsBlankToGlobal(t *testing.T) {
	events := []event.Event{
		{Project: "beta"},
		{Project: ""},
		{Project: "alpha"},
		{Project: "beta"},
	}

	got := chronicleProjectOptions(events)
	want := []string{"alpha", "beta", defaultProjectName}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected project options\nwant: %#v\n got: %#v", want, got)
	}
}

func TestChronicleDisplayHelpers(t *testing.T) {
	if got := chronicleScopeLabel(""); got != "all projects" {
		t.Fatalf("unexpected scope label for empty project: %q", got)
	}
	if got := chronicleScopeLabel(defaultProjectName); got != "global" {
		t.Fatalf("unexpected scope label for default project: %q", got)
	}
	if got := chronicleScopeLabel("alpha"); got != "alpha" {
		t.Fatalf("unexpected scope label for named project: %q", got)
	}

	if got := chronicleCountLabel(1); got != "1 entry" {
		t.Fatalf("unexpected count label for singular: %q", got)
	}
	if got := chronicleCountLabel(2); got != "2 entries" {
		t.Fatalf("unexpected count label for plural: %q", got)
	}

	if got := chroniclePreviewTitle(nil); got != "No entry selected" {
		t.Fatalf("unexpected nil preview title: %q", got)
	}
	if got := chroniclePreviewTitle(&event.Event{Title: "   "}); got != "(untitled)" {
		t.Fatalf("unexpected untitled preview title: %q", got)
	}
	if got := chroniclePreviewTitle(&event.Event{Title: "Keep this"}); got != "Keep this" {
		t.Fatalf("unexpected populated preview title: %q", got)
	}
}

func TestChronicleBodyExcerpt_EmptyAndLimited(t *testing.T) {
	if got := chronicleBodyExcerpt("\n\n", 4); got != "(no notes yet)" {
		t.Fatalf("unexpected empty body excerpt: %q", got)
	}

	body := "a\nb\nc\nd\ne"
	got := chronicleBodyExcerpt(body, 3)
	want := "a\nb\nc\n..."
	if got != want {
		t.Fatalf("unexpected limited excerpt\nwant: %q\n got: %q", want, got)
	}
}

func TestChronicleDaySummary_ValidAndFallback(t *testing.T) {
	if got := chronicleDaySummary("bad-date", 2); got != "2 entries" {
		t.Fatalf("unexpected fallback day summary: %q", got)
	}

	got := chronicleDaySummary("2026-04-20", 1)
	if !strings.Contains(got, "Monday, April 20") || !strings.Contains(got, "1 entry") {
		t.Fatalf("unexpected parsed day summary: %q", got)
	}
}
