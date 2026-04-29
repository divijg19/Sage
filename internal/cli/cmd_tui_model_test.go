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
	if m.statusTone != chronicleStatusInfo {
		t.Fatalf("unexpected initial status tone: %q", m.statusTone)
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
	if m.inputMode != chronicleInputSearch {
		t.Fatalf("expected search input mode by default, got %q", m.inputMode)
	}
}

func TestChronicleModel_BreakpointsAndDimensions(t *testing.T) {
	compact := chronicleModel{width: 80, height: 30}
	if !compact.isCompact() || compact.isMedium() {
		t.Fatalf("expected compact breakpoint for width=80")
	}
	if got := compact.timelineWidth(); got != 80 {
		t.Fatalf("unexpected compact timeline width: %d", got)
	}
	if got := compact.timelineHeight(); got != 18 {
		t.Fatalf("unexpected compact timeline height: %d", got)
	}

	medium := chronicleModel{width: 100, height: 30}
	if medium.isCompact() || !medium.isMedium() {
		t.Fatalf("expected medium breakpoint for width=100")
	}
	if got := medium.timelineWidth(); got != 70 {
		t.Fatalf("unexpected medium timeline width: %d", got)
	}
	if got := medium.timelineHeight(); got != 10 {
		t.Fatalf("unexpected medium timeline height: %d", got)
	}

	wide := chronicleModel{width: 130, height: 30}
	if wide.isCompact() || wide.isMedium() {
		t.Fatalf("expected wide breakpoint for width=130")
	}
	if got := wide.timelineWidth(); got != 54 {
		t.Fatalf("unexpected wide timeline width: %d", got)
	}
	if got := wide.timelineHeight(); got != 18 {
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
	m.filterIndex = 1 + len(m.projects) // first kind row when there are no projects

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

	m.filterIndex = 0
	next := m.updateFilterPalette(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if next.selectedProject != "" {
		t.Fatalf("expected all-projects toggle to clear selected project, got %q", next.selectedProject)
	}

	next.filterIndex = 1 + len(next.projects) + 3
	next = next.updateFilterPalette(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !next.tagFilter["auth"] {
		t.Fatalf("expected tag toggle to enable auth")
	}
	if next.status != "Added #auth filter" {
		t.Fatalf("unexpected tag enable status: %q", next.status)
	}

	next = next.updateFilterPalette(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if next.tagFilter["auth"] {
		t.Fatalf("expected second tag toggle to disable auth")
	}
}

func TestUpdateSearch_EnterAndEscape(t *testing.T) {
	m := fixtureChronicleModel(100, 24)
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
	if afterEnter.status != `Search applied: "durability"` {
		t.Fatalf("unexpected enter status: %q", afterEnter.status)
	}

	afterEnter.focused = "search"
	afterEnter.queryInput.SetValue("  stale cache  ")
	model, _ = afterEnter.updateSearch(tea.KeyMsg{Type: tea.KeyEsc})
	afterEsc := model.(chronicleModel)
	if afterEsc.status != `Search applied: "stale cache"` {
		t.Fatalf("unexpected esc status: %q", afterEsc.status)
	}
}

func TestBottomInput_SearchAndModeToggle(t *testing.T) {
	m := fixtureChronicleModel(100, 24)

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	afterSlash := model.(chronicleModel)
	if afterSlash.focused != "input" || afterSlash.inputMode != chronicleInputSearch {
		t.Fatalf("expected slash to focus search input, focused=%q mode=%q", afterSlash.focused, afterSlash.inputMode)
	}

	model, _ = afterSlash.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	typing := model.(chronicleModel)
	if typing.query != "a" {
		t.Fatalf("expected live search query to update, got %q", typing.query)
	}

	model, _ = typing.Update(tea.KeyMsg{Type: tea.KeyTab})
	commandMode := model.(chronicleModel)
	if commandMode.inputMode != chronicleInputCommand {
		t.Fatalf("expected tab to switch bottom input to command mode")
	}

	model, _ = commandMode.Update(tea.KeyMsg{Type: tea.KeyEsc})
	closed := model.(chronicleModel)
	if closed.focused != "" {
		t.Fatalf("expected escape to close bottom input")
	}
}

func TestBottomInput_CommandExecution(t *testing.T) {
	m := fixtureChronicleModel(100, 24)

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	commandMode := model.(chronicleModel)
	commandMode.commandInput.SetValue("view 2")

	model, _ = commandMode.Update(tea.KeyMsg{Type: tea.KeyEnter})
	viewing := model.(chronicleModel)
	if viewing.focused != "" {
		t.Fatalf("expected command enter to close bottom input")
	}
	selected := viewing.selected()
	if selected == nil || selected.Kind != chronicleRowEntry || selected.Event.Seq != 2 {
		t.Fatalf("expected view command to select entry 2, got %#v", selected)
	}

	viewing.commandInput.SetValue("clear")
	viewing.query = "auth"
	viewing.queryInput.SetValue("auth")
	viewing.tagFilter["auth"] = true
	model, _ = viewing.runCommandInput()
	cleared := model.(chronicleModel)
	if cleared.query != "" || cleared.queryInput.Value() != "" || len(cleared.tagFilter) != 0 {
		t.Fatalf("expected clear command to reset search and tag filters")
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
	if len(lines) != 7 {
		t.Fatalf("expected 7 lines (title, meta, excerpt), got %d", len(lines))
	}

	meta := ansi.Strip(lines[1])
	if !strings.Contains(meta, "#alpha #zeta") {
		t.Fatalf("expected sorted tags in meta line, got %q", meta)
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
	theme := newChronicleTheme()
	m := newChronicleModel(chronicleOptions{})
	m.rows = []chronicleRow{{
		Kind:     chronicleRowDay,
		DayLabel: "Mon, Apr 20 2026",
		DayKey:   "2026-04-20",
		DayCount: 2,
	}}
	m.selectedRow = 0

	dayPreview := ansi.Strip(m.renderPreview(theme, 60, 14))
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

	entryPreview := ansi.Strip(m.renderPreview(theme, 60, 14))
	if !strings.Contains(entryPreview, "(untitled)") {
		t.Fatalf("expected untitled fallback in entry preview:\n%s", entryPreview)
	}
	if !strings.Contains(entryPreview, "Project: global") {
		t.Fatalf("expected global project label in entry preview:\n%s", entryPreview)
	}
	if !strings.Contains(entryPreview, "#alpha") || !strings.Contains(entryPreview, "#zeta") {
		t.Fatalf("expected sorted tags in entry preview:\n%s", entryPreview)
	}
}

func TestView_CompactHintsAndOverlay(t *testing.T) {
	m := fixtureChronicleModel(80, 24)
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "tab") || !strings.Contains(view, "inspect") {
		t.Fatalf("expected compact-mode inspect hint")
	}

	m.showFilters = true
	view = ansi.Strip(m.View())
	if !strings.Contains(view, "Filter Chronicle") {
		t.Fatalf("expected filter overlay to be rendered")
	}
}

func TestCompactTabModeAndScrollVisibility(t *testing.T) {
	m := fixtureChronicleModel(80, 24)

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	afterTab := model.(chronicleModel)
	if !afterTab.showPreview {
		t.Fatalf("expected compact tab to switch to inspector mode")
	}
	if afterTab.status != "Inspector mode" {
		t.Fatalf("unexpected compact tab status: %q", afterTab.status)
	}

	scrollModel := fixtureChronicleScrollModel(80, 16)
	for i := 0; i < len(scrollModel.rows)-1; i++ {
		scrollModel.moveSelection(1)
	}
	if scrollModel.scrollLine == 0 {
		t.Fatalf("expected scroll line to advance for later rows")
	}
}

func TestQuickEntryValidationAndCancel(t *testing.T) {
	m := fixtureChronicleModel(100, 24)
	m.openQuickEntry()
	m.quickField = 2

	model, _ := m.updateQuickEntry(tea.KeyMsg{Type: tea.KeyEnter})
	afterEnter := *(model.(*chronicleModel))
	if afterEnter.status != "Quick entry needs a title" {
		t.Fatalf("unexpected validation status: %q", afterEnter.status)
	}
	if afterEnter.statusTone != chronicleStatusWarn {
		t.Fatalf("expected warning tone after validation failure")
	}

	model, _ = afterEnter.updateQuickEntry(tea.KeyMsg{Type: tea.KeyEsc})
	afterEsc := model.(chronicleModel)
	if afterEsc.showQuick {
		t.Fatalf("expected quick entry to close on escape")
	}
	if afterEsc.status != "Quick entry canceled" {
		t.Fatalf("unexpected cancel status: %q", afterEsc.status)
	}
}

func fixtureChronicleModel(width int, height int) chronicleModel {
	events := []event.Event{
		{
			Seq:       1,
			ID:        "evt-1",
			Timestamp: time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC),
			Project:   "alpha",
			Kind:      event.RecordKind,
			Title:     "Investigate auth cache",
			Content:   "Trace request path\nVerify token invalidation\nCapture failing reproduction",
			Tags:      []string{"auth", "backend"},
		},
		{
			Seq:       2,
			ID:        "evt-2",
			Timestamp: time.Date(2026, 4, 22, 11, 30, 0, 0, time.UTC),
			Project:   "alpha",
			Kind:      event.DecisionKind,
			Title:     "Switch to sqlite wal",
			Content:   "Durability first\nRoll out on developer machines\nDocument rollback plan",
			Tags:      []string{"storage"},
		},
		{
			Seq:       3,
			ID:        "evt-3",
			Timestamp: time.Date(2026, 4, 23, 8, 45, 0, 0, time.UTC),
			Project:   "",
			Kind:      event.CommitKind,
			Title:     "Polish chronicle layout",
			Content:   "Tighten spacing\nNormalize chips\nAdjust inspector copy",
			Tags:      []string{"ux", "release"},
		},
	}

	m := newChronicleModel(chronicleOptions{})
	m.width = width
	m.height = height
	m.loading = false
	m.events = events
	m.availableTags = chronicleUnionTags(nil, events)
	m.projects = chronicleProjectOptions(events)
	m.status = "Ready"
	m.statusTone = chronicleStatusInfo
	m.rebuildRows(0)
	return m
}

func fixtureChronicleScrollModel(width int, height int) chronicleModel {
	events := make([]event.Event, 0, 8)
	base := time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)
	for i := 0; i < 8; i++ {
		events = append(events, event.Event{
			Seq:       int64(i + 1),
			ID:        "evt-scroll",
			Timestamp: base.Add(time.Duration(i) * time.Hour),
			Project:   "alpha",
			Kind:      event.RecordKind,
			Title:     "Scrollable note",
			Content:   "Context line one\nContext line two\nContext line three",
			Tags:      []string{"auth"},
		})
	}

	m := newChronicleModel(chronicleOptions{})
	m.width = width
	m.height = height
	m.loading = false
	m.events = events
	m.availableTags = chronicleUnionTags(nil, events)
	m.projects = chronicleProjectOptions(events)
	m.status = "Ready"
	m.statusTone = chronicleStatusInfo
	m.rebuildRows(0)
	return m
}
