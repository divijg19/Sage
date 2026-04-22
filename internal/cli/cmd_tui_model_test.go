package cli

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/divijg19/sage/internal/event"
)

func TestNewChronicleModel_SeedsState(t *testing.T) {
	m := newChronicleModel(chronicleOptions{
		Query:   "auth cache",
		Project: "alpha",
		Tags:    []string{"auth", "backend"},
	})

	if m.loading != true {
		t.Fatalf("expected model to begin loading")
	}
	if m.status != "Loading Chronicle..." {
		t.Fatalf("unexpected initial status: %q", m.status)
	}
	if m.selectedProject != "alpha" {
		t.Fatalf("unexpected selected project: %q", m.selectedProject)
	}
	if m.queryInput.Value() != "auth cache" {
		t.Fatalf("unexpected query input value: %q", m.queryInput.Value())
	}
	if !m.tagFilter["auth"] || !m.tagFilter["backend"] {
		t.Fatalf("expected initial tag filter to include provided tags")
	}
	if m.quickKind != event.RecordKind {
		t.Fatalf("unexpected initial quick kind: %q", m.quickKind)
	}
}

func TestChronicleModel_BreakpointsAndDimensions(t *testing.T) {
	compact := chronicleModel{width: 80, height: 30}
	if !compact.isCompact() || compact.isMedium() {
		t.Fatalf("expected compact breakpoint for width=80")
	}
	if got := compact.timelineWidth(); got != 78 {
		t.Fatalf("unexpected compact timeline width: %d", got)
	}
	if got := compact.timelineHeight(); got != 26 {
		t.Fatalf("unexpected compact timeline height: %d", got)
	}

	medium := chronicleModel{width: 100, height: 30}
	if medium.isCompact() || !medium.isMedium() {
		t.Fatalf("expected medium breakpoint for width=100")
	}
	if got := medium.timelineWidth(); got != 71 {
		t.Fatalf("unexpected medium timeline width: %d", got)
	}
	if got := medium.timelineHeight(); got != 15 {
		t.Fatalf("unexpected medium timeline height: %d", got)
	}

	wide := chronicleModel{width: 130, height: 30}
	if wide.isCompact() || wide.isMedium() {
		t.Fatalf("expected wide breakpoint for width=130")
	}
	if got := wide.timelineWidth(); got != 56 {
		t.Fatalf("unexpected wide timeline width: %d", got)
	}
	if got := wide.timelineHeight(); got != 26 {
		t.Fatalf("unexpected wide timeline height: %d", got)
	}
}

func TestUpdateFilterPalette_KindCannotDisableAll(t *testing.T) {
	m := newChronicleModel(chronicleOptions{})
	m.events = []event.Event{
		{
			Seq:       1,
			Timestamp: time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC),
			Kind:      event.RecordKind,
			Title:     "one",
		},
	}
	m.kindFilter = map[event.EntryKind]bool{
		event.RecordKind:   true,
		event.DecisionKind: false,
		event.CommitKind:   false,
	}
	m.filterIndex = 1 // first kind row when there are no projects

	next := m.updateFilterPalette(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !next.kindFilter[event.RecordKind] {
		t.Fatalf("expected record kind to stay enabled when it is the last enabled kind")
	}
}

func TestUpdateFilterPalette_TogglesScopeAndTags(t *testing.T) {
	m := newChronicleModel(chronicleOptions{})
	m.events = []event.Event{
		{Seq: 1, Timestamp: time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC), Project: "alpha", Kind: event.RecordKind, Title: "one", Tags: []string{"auth"}},
	}
	m.projects = []string{"alpha", "beta"}
	m.availableTags = []string{"auth"}
	m.selectedProject = "alpha"

	m.filterIndex = 0 // all projects
	next := m.updateFilterPalette(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if next.selectedProject != "" {
		t.Fatalf("expected all-projects toggle to clear selected project, got %q", next.selectedProject)
	}

	next.filterIndex = 1 + len(next.projects) + 3 // first tag row
	next = next.updateFilterPalette(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !next.tagFilter["auth"] {
		t.Fatalf("expected tag toggle to enable auth")
	}

	next = next.updateFilterPalette(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if next.tagFilter["auth"] {
		t.Fatalf("expected second tag toggle to disable auth")
	}
}

func TestUpdateSearch_EnterAndEscape(t *testing.T) {
	m := newChronicleModel(chronicleOptions{})
	m.focused = "search"
	m.queryInput.SetValue("durability")

	model, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeyEnter})
	afterEnter := model.(chronicleModel)
	if afterEnter.focused != "" {
		t.Fatalf("expected search focus to clear on enter")
	}
	if afterEnter.query != "durability" {
		t.Fatalf("expected query to persist, got %q", afterEnter.query)
	}

	afterEnter.focused = "search"
	afterEnter.queryInput.SetValue("  stale cache  ")
	model, _ = afterEnter.updateSearch(tea.KeyMsg{Type: tea.KeyEsc})
	afterEsc := model.(chronicleModel)
	if afterEsc.status != "Search cleared to \"stale cache\"" {
		t.Fatalf("unexpected esc status: %q", afterEsc.status)
	}
}

func TestRenderTimelineRow_EntryOpenShowsSortedTagsAndExcerpt(t *testing.T) {
	m := newChronicleModel(chronicleOptions{})
	row := chronicleRow{
		Kind:      chronicleRowEntry,
		EntryOpen: true,
		Event: event.Event{
			Seq:       7,
			Timestamp: time.Date(2026, 4, 20, 15, 4, 0, 0, time.UTC),
			Kind:      event.DecisionKind,
			Title:     "Investigate drift",
			Tags:      []string{"zeta", "alpha"},
		},
		PreviewBody: "one\ntwo\nthree\nfour\nfive",
	}

	lines := m.renderTimelineRow(row, false, 80)
	if len(lines) != 6 {
		t.Fatalf("expected 6 lines (header + 5 excerpt lines), got %d", len(lines))
	}

	head := ansi.Strip(lines[0])
	if !strings.Contains(head, "#alpha #zeta") {
		t.Fatalf("expected sorted tags in head line, got %q", head)
	}
	if !strings.Contains(ansi.Strip(lines[len(lines)-1]), "...") {
		t.Fatalf("expected excerpt to include ellipsis line")
	}

	for i, line := range lines {
		if w := ansi.StringWidth(line); w > 80 {
			t.Fatalf("line %d exceeds width 80: width=%d", i, w)
		}
	}
}

func TestRenderPreview_DayAndEntry(t *testing.T) {
	m := newChronicleModel(chronicleOptions{})
	m.rows = []chronicleRow{{
		Kind:     chronicleRowDay,
		DayLabel: "Mon, Apr 20 2026",
		DayKey:   "2026-04-20",
		DayCount: 2,
	}}
	m.selectedRow = 0

	dayPreview := ansi.Strip(m.renderPreview(60, 14))
	if !strings.Contains(dayPreview, "Mon, Apr 20 2026") || !strings.Contains(dayPreview, "2 entries") {
		t.Fatalf("unexpected day preview:\n%s", dayPreview)
	}

	m.rows = []chronicleRow{{
		Kind: chronicleRowEntry,
		Event: event.Event{
			Seq:       9,
			Timestamp: time.Date(2026, 4, 20, 15, 4, 0, 0, time.UTC),
			Project:   "",
			Kind:      event.DecisionKind,
			Title:     "",
			Tags:      []string{"zeta", "alpha"},
			Content:   "a\nb\nc\nd\ne",
		},
	}}
	m.selectedRow = 0

	entryPreview := ansi.Strip(m.renderPreview(60, 14))
	if !strings.Contains(entryPreview, "(untitled)") {
		t.Fatalf("expected untitled fallback in entry preview:\n%s", entryPreview)
	}
	if !strings.Contains(entryPreview, "Project: global") {
		t.Fatalf("expected global project label in entry preview:\n%s", entryPreview)
	}
	if !strings.Contains(entryPreview, "Tags: #alpha #zeta") {
		t.Fatalf("expected sorted tags in entry preview:\n%s", entryPreview)
	}
}

func TestView_CompactHintsAndOverlay(t *testing.T) {
	m := newChronicleModel(chronicleOptions{})
	m.width = 80
	m.height = 24
	m.status = "ready"

	view := ansi.Strip(m.View())
	if !strings.Contains(view, "tab preview") {
		t.Fatalf("expected compact-mode tab preview hint")
	}

	m.showFilters = true
	view = ansi.Strip(m.View())
	if !strings.Contains(view, "Filter Chronicle") {
		t.Fatalf("expected filter overlay to be rendered")
	}
}
