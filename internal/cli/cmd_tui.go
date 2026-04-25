package cli

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"

	"github.com/divijg19/sage/internal/entryflow"
	"github.com/divijg19/sage/internal/event"
)

var (
	tuiAll     bool
	tuiProject string
	tuiTags    []string
	tuiQuery   string
)

type chronicleProgram interface {
	Run() (tea.Model, error)
}

var newChronicleProgram = func(model tea.Model) chronicleProgram {
	return tea.NewProgram(model, tea.WithAltScreen())
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the Chronicle terminal interface",
	Long: "Open Chronicle, Sage's keyboard-first terminal interface for browsing\n" +
		"and adding entries from the same local event store used by the CLI.",
	RunE: func(cmd *cobra.Command, args []string) error {
		model := newChronicleModel(chronicleOptionsFromFlags())
		program := newChronicleProgram(model)
		_, err := program.Run()
		return err
	},
}

func init() {
	tuiCmd.Flags().BoolVar(&tuiAll, "all", false, "show entries from all projects")
	tuiCmd.Flags().StringVar(&tuiProject, "project", "", "override project scope (ignores active project)")
	tuiCmd.Flags().StringArrayVar(&tuiTags, "tags", nil, "filter by tags (repeatable or comma-separated)")
	tuiCmd.Flags().StringVar(&tuiQuery, "query", "", "apply an initial text query")
	rootCmd.AddCommand(tuiCmd)
}

type chronicleOptions struct {
	Query   string
	Project string
	Tags    []string
}

func chronicleOptionsFromFlags() chronicleOptions {
	project, filter := resolveProjectFilter(tuiProject, tuiAll)
	if !filter {
		project = ""
	}
	return chronicleOptions{
		Query:   strings.TrimSpace(tuiQuery),
		Project: project,
		Tags:    parseTags(tuiTags),
	}
}

type chronicleDataLoadedMsg struct {
	events    []event.Event
	tags      []string
	highlight int64
	err       error
}

type chronicleEditorFinishedMsg struct {
	err error
}

type chronicleStatusTone string

const (
	chronicleStatusInfo    chronicleStatusTone = "info"
	chronicleStatusSuccess chronicleStatusTone = "success"
	chronicleStatusWarn    chronicleStatusTone = "warn"
	chronicleStatusError   chronicleStatusTone = "error"
)

type chronicleModel struct {
	width  int
	height int

	queryInput textinput.Model
	titleInput textinput.Model
	tagsInput  textinput.Model

	events          []event.Event
	rows            []chronicleRow
	projects        []string
	availableTags   []string
	selectedProject string
	tagFilter       map[string]bool
	kindFilter      map[event.EntryKind]bool
	collapsedDays   map[string]bool
	expandedEntries map[int64]bool

	selectedRow int
	scrollLine  int

	query       string
	focused     string
	status      string
	statusTone  chronicleStatusTone
	loading     bool
	showFilters bool
	showQuick   bool
	showPreview bool
	filterIndex int
	quickField  int
	quickKind   event.EntryKind

	pending *chroniclePendingEditor
}

type chroniclePendingEditor struct {
	launch  *editorLaunch
	request entryflow.FinalizeRequest
}

type chronicleFilterItem struct {
	Group string
	Label string
	Kind  string
	Value string
	On    bool
}

func newChronicleModel(opts chronicleOptions) chronicleModel {
	queryInput := textinput.New()
	queryInput.Prompt = ""
	queryInput.Placeholder = "Search title, notes, tags, project"
	queryInput.SetValue(opts.Query)
	queryInput.CharLimit = 256
	queryInput.Width = 26

	titleInput := textinput.New()
	titleInput.Prompt = ""
	titleInput.Placeholder = "Entry title"
	titleInput.CharLimit = 120
	titleInput.Width = 30

	tagsInput := textinput.New()
	tagsInput.Prompt = ""
	tagsInput.Placeholder = "auth,backend"
	tagsInput.CharLimit = 120
	tagsInput.Width = 30

	kindFilter := map[event.EntryKind]bool{
		event.RecordKind:   true,
		event.DecisionKind: true,
		event.CommitKind:   true,
	}

	tagFilter := make(map[string]bool)
	for _, tag := range opts.Tags {
		tagFilter[tag] = true
	}

	return chronicleModel{
		queryInput:      queryInput,
		titleInput:      titleInput,
		tagsInput:       tagsInput,
		selectedProject: opts.Project,
		tagFilter:       tagFilter,
		kindFilter:      kindFilter,
		collapsedDays:   map[string]bool{},
		expandedEntries: map[int64]bool{},
		query:           opts.Query,
		loading:         true,
		quickKind:       event.RecordKind,
		status:          "Loading Chronicle...",
		statusTone:      chronicleStatusInfo,
	}
}

func (m chronicleModel) Init() tea.Cmd {
	return tea.Batch(loadChronicleDataCmd(), textinput.Blink)
}

func (m chronicleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.queryInput.Width = max(18, min(34, m.width-16))
		return m, nil

	case chronicleDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.setStatusError(msg.err.Error())
			return m, nil
		}
		m.events = msg.events
		m.availableTags = msg.tags
		m.projects = chronicleProjectOptions(msg.events)
		m.rebuildRows(msg.highlight)
		m.setStatusInfo(m.scopeStatusMessage())
		return m, nil

	case chronicleEditorFinishedMsg:
		return m.finishQuickEntry(msg.err)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.pending != nil {
			return m, nil
		}

		if m.showQuick {
			return m.updateQuickEntry(msg)
		}

		if m.showFilters {
			return m.updateFilterPalette(msg), nil
		}

		if m.focused == "search" {
			return m.updateSearch(msg)
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "j", "down":
			m.moveSelection(1)
		case "k", "up":
			m.moveSelection(-1)
		case "enter", " ":
			m.toggleSelected()
		case "/":
			m.focused = "search"
			m.setStatusInfo("Search Chronicle")
			return m, m.queryInput.Focus()
		case "f":
			m.showFilters = true
			m.filterIndex = 0
			m.setStatusInfo("Filter Chronicle")
		case "n":
			m.openQuickEntry()
			return m, m.titleInput.Focus()
		case "r":
			m.loading = true
			m.setStatusInfo("Reloading Chronicle...")
			return m, loadChronicleDataCmd()
		case "tab":
			if m.isCompact() {
				m.showPreview = !m.showPreview
				if m.showPreview {
					m.setStatusInfo("Inspector mode")
				} else {
					m.setStatusInfo("Browse mode")
				}
			}
		}

		return m, nil
	}

	return m, nil
}

func (m chronicleModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.focused = ""
		m.queryInput.Blur()
		m.query = strings.TrimSpace(m.queryInput.Value())
		m.queryInput.SetValue(m.query)
		m.rebuildRows(0)
		m.setStatusInfo(chronicleSearchStatus(m.query))
		return m, nil
	case "enter":
		m.focused = ""
		m.queryInput.Blur()
		m.query = strings.TrimSpace(m.queryInput.Value())
		m.queryInput.SetValue(m.query)
		m.rebuildRows(0)
		m.setStatusInfo(chronicleSearchStatus(m.query))
		return m, nil
	}

	var cmd tea.Cmd
	m.queryInput, cmd = m.queryInput.Update(msg)
	m.query = strings.TrimSpace(m.queryInput.Value())
	m.rebuildRows(0)
	m.setStatusInfo(chronicleSearchDraftStatus(m.query))
	return m, cmd
}

func (m chronicleModel) updateQuickEntry(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeQuickEntry("Quick entry canceled", chronicleStatusWarn)
		return m, nil
	case "tab":
		m.quickField = (m.quickField + 1) % 3
		return m, m.focusQuickField()
	case "shift+tab":
		m.quickField--
		if m.quickField < 0 {
			m.quickField = 2
		}
		return m, m.focusQuickField()
	case "up":
		if m.quickField > 0 {
			m.quickField--
		}
		return m, m.focusQuickField()
	case "down":
		if m.quickField < 2 {
			m.quickField++
		}
		return m, m.focusQuickField()
	case "left", "h":
		if m.quickField == 2 {
			m.toggleQuickKind()
			return m, nil
		}
	case "right", "l":
		if m.quickField == 2 {
			m.toggleQuickKind()
			return m, nil
		}
	case "enter":
		if m.quickField < 2 {
			m.quickField++
			return m, m.focusQuickField()
		}
		return m.startQuickEntry()
	}

	var cmd tea.Cmd
	switch m.quickField {
	case 0:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case 1:
		m.tagsInput, cmd = m.tagsInput.Update(msg)
	}
	return m, cmd
}

func (m chronicleModel) updateFilterPalette(msg tea.KeyMsg) chronicleModel {
	items := m.filterItems()
	if len(items) == 0 {
		m.showFilters = false
		return m
	}
	if m.filterIndex >= len(items) {
		m.filterIndex = len(items) - 1
	}

	switch msg.String() {
	case "esc":
		m.showFilters = false
		m.setStatusInfo("Closed filters")
		return m
	case "j", "down":
		if m.filterIndex < len(items)-1 {
			m.filterIndex++
		}
	case "k", "up":
		if m.filterIndex > 0 {
			m.filterIndex--
		}
	case " ", "enter":
		item := items[m.filterIndex]
		switch item.Kind {
		case "scope_all":
			m.selectedProject = ""
			m.setStatusInfo("Scope set to all projects")
		case "scope_project":
			m.selectedProject = item.Value
			m.setStatusInfo("Scope set to " + chronicleScopeLabel(item.Value))
		case "kind":
			kind := event.EntryKind(item.Value)
			m.kindFilter[kind] = !m.kindFilter[kind]
			if !m.hasEnabledKinds() {
				m.kindFilter[kind] = true
			}
			m.setStatusInfo("Kinds updated")
		case "tag":
			if m.tagFilter[item.Value] {
				delete(m.tagFilter, item.Value)
				m.setStatusInfo("Removed #" + item.Value + " filter")
			} else {
				m.tagFilter[item.Value] = true
				m.setStatusInfo("Added #" + item.Value + " filter")
			}
		}
		m.rebuildRows(0)
	}

	return m
}

func (m chronicleModel) finishQuickEntry(execErr error) (tea.Model, tea.Cmd) {
	if m.pending == nil {
		return m, nil
	}

	launch := m.pending.launch
	req := m.pending.request
	m.pending = nil
	defer launch.cleanup()

	edited := ""
	if execErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(execErr, &exitErr) {
			m.setStatusError(execErr.Error())
			return m, nil
		}
	} else {
		content, err := launch.result()
		if err != nil {
			m.setStatusError(err.Error())
			return m, nil
		}
		edited = content
	}

	s, err := openGlobalStore()
	if err != nil {
		m.setStatusError(err.Error())
		return m, nil
	}

	req.Edited = edited
	result, err := entryflow.Finalize(req, entryflow.Dependencies{
		Store:       s,
		EnsureTags:  ensureTagsConfigured,
		ResolveKind: resolveKind,
	})
	if err != nil {
		m.setStatusError(err.Error())
		return m, nil
	}

	switch result.Status {
	case entryflow.StatusSaved:
		m.setStatusSuccess("Entry recorded")
		m.showQuick = false
		m.focused = ""
		return m, loadChronicleDataCmdWithHighlight(result.Event.Seq)
	case entryflow.StatusCanceled:
		m.setStatusWarn("Editor canceled")
	case entryflow.StatusUnchanged:
		m.setStatusInfo("No changes recorded")
	case entryflow.StatusEmpty:
		m.setStatusWarn("Entry was empty")
	case entryflow.StatusDuplicate:
		m.setStatusWarn("Duplicate entry skipped")
	}

	m.showQuick = false
	m.focused = ""
	return m, loadChronicleDataCmd()
}

func (m *chronicleModel) startQuickEntry() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(m.titleInput.Value())
	if title == "" {
		m.setStatusWarn("Quick entry needs a title")
		return m, nil
	}

	explicitKind := "record"
	if m.quickKind == event.DecisionKind {
		explicitKind = "decision"
	}

	tags := parseTags([]string{m.tagsInput.Value()})
	prepared := entryflow.PrepareInitialBuffer(title, explicitKind, "", "")
	launch, err := prepareEditorLaunch(prepared.Body)
	if err != nil {
		m.setStatusError(err.Error())
		return m, nil
	}

	m.pending = &chroniclePendingEditor{
		launch: launch,
		request: entryflow.FinalizeRequest{
			Title:         title,
			ExplicitKind:  explicitKind,
			SuggestedKind: "",
			SeedKind:      prepared.SeedKind,
			InitialBody:   prepared.Body,
			Project:       projectForNewEntry(),
			Tags:          tags,
		},
	}
	m.setStatusInfo("Opening editor...")
	return m, tea.ExecProcess(launch.command(), func(err error) tea.Msg {
		return chronicleEditorFinishedMsg{err: err}
	})
}

func (m *chronicleModel) openQuickEntry() {
	m.showQuick = true
	m.quickField = 0
	m.quickKind = event.RecordKind
	m.focused = "quick"
	m.titleInput.SetValue("")
	m.tagsInput.SetValue("")
	m.setStatusInfo("Quick entry")
}

func (m *chronicleModel) closeQuickEntry(status string, tone chronicleStatusTone) {
	m.showQuick = false
	m.focused = ""
	m.titleInput.Blur()
	m.tagsInput.Blur()
	m.setStatus(status, tone)
}

func (m *chronicleModel) focusQuickField() tea.Cmd {
	m.titleInput.Blur()
	m.tagsInput.Blur()
	switch m.quickField {
	case 0:
		return m.titleInput.Focus()
	case 1:
		return m.tagsInput.Focus()
	default:
		return nil
	}
}

func (m *chronicleModel) toggleQuickKind() {
	if m.quickKind == event.DecisionKind {
		m.quickKind = event.RecordKind
	} else {
		m.quickKind = event.DecisionKind
	}
	m.setStatusInfo("Quick entry kind: " + chronicleKindLabel(m.quickKind))
}

func (m chronicleModel) quickKindLabel() string {
	return chronicleKindLabel(m.quickKind)
}

func (m *chronicleModel) moveSelection(delta int) {
	if len(m.rows) == 0 {
		return
	}
	m.selectedRow += delta
	if m.selectedRow < 0 {
		m.selectedRow = 0
	}
	if m.selectedRow >= len(m.rows) {
		m.selectedRow = len(m.rows) - 1
	}
	m.ensureSelectedVisible()
}

func (m *chronicleModel) toggleSelected() {
	row := m.selected()
	if row == nil {
		return
	}
	switch row.Kind {
	case chronicleRowDay:
		m.collapsedDays[row.DayKey] = !m.collapsedDays[row.DayKey]
		m.rebuildRows(0)
		if m.collapsedDays[row.DayKey] {
			m.setStatusInfo("Day collapsed")
		} else {
			m.setStatusInfo("Day expanded")
		}
	case chronicleRowEntry:
		m.expandedEntries[row.Event.Seq] = !m.expandedEntries[row.Event.Seq]
		m.rebuildRows(row.Event.Seq)
		if m.expandedEntries[row.Event.Seq] {
			m.setStatusInfo("Entry expanded")
		} else {
			m.setStatusInfo("Entry collapsed")
		}
	}
}

func (m *chronicleModel) rebuildRows(preferSeq int64) {
	m.rows = buildChronicleRows(filterChronicleEvents(m.events, m.filters()), m.collapsedDays, m.expandedEntries)
	if len(m.rows) == 0 {
		m.selectedRow = 0
		m.scrollLine = 0
		return
	}

	previousDay := ""
	var previousSeq int64
	if current := m.selected(); current != nil {
		previousDay = current.DayKey
		if current.Kind == chronicleRowEntry {
			previousSeq = current.Event.Seq
		}
	}

	target := 0
	if preferSeq != 0 {
		for i, row := range m.rows {
			if row.Kind == chronicleRowEntry && row.Event.Seq == preferSeq {
				target = i
				break
			}
		}
	} else if previousSeq != 0 {
		for i, row := range m.rows {
			if row.Kind == chronicleRowEntry && row.Event.Seq == previousSeq {
				target = i
				break
			}
		}
	} else if previousDay != "" {
		for i, row := range m.rows {
			if row.DayKey == previousDay {
				target = i
				break
			}
		}
	}

	m.selectedRow = target
	m.ensureSelectedVisible()
}

func (m *chronicleModel) ensureSelectedVisible() {
	height := m.timelineHeight()
	if height <= 0 || len(m.rows) == 0 {
		m.scrollLine = 0
		return
	}

	width := max(24, m.timelineWidth()-4)
	start := 0
	selectedStart := 0
	selectedEnd := 0
	for i, row := range m.rows {
		h := len(m.renderTimelineRow(row, i == m.selectedRow, width))
		if i == m.selectedRow {
			selectedStart = start
			selectedEnd = start + h
			break
		}
		start += h
	}

	if selectedStart < m.scrollLine {
		m.scrollLine = selectedStart
	}
	if selectedEnd > m.scrollLine+height {
		m.scrollLine = selectedEnd - height
	}
	if m.scrollLine < 0 {
		m.scrollLine = 0
	}
}

func (m chronicleModel) filters() chronicleFilters {
	enabledKinds := map[event.EntryKind]bool{}
	for kind, on := range m.kindFilter {
		if on {
			enabledKinds[kind] = true
		}
	}

	return chronicleFilters{
		Query:        m.query,
		Project:      m.selectedProject,
		EnabledKinds: enabledKinds,
		EnabledTags:  m.tagFilter,
	}
}

func (m chronicleModel) selected() *chronicleRow {
	if len(m.rows) == 0 || m.selectedRow < 0 || m.selectedRow >= len(m.rows) {
		return nil
	}
	row := m.rows[m.selectedRow]
	return &row
}

func (m chronicleModel) selectedEvent() *event.Event {
	row := m.selected()
	if row == nil {
		return nil
	}
	if row.Kind == chronicleRowEntry {
		return &row.Event
	}
	return nil
}

func (m chronicleModel) hasEnabledKinds() bool {
	for _, on := range m.kindFilter {
		if on {
			return true
		}
	}
	return false
}

func (m chronicleModel) filterItems() []chronicleFilterItem {
	items := []chronicleFilterItem{
		{Group: "Scope", Label: "All projects", Kind: "scope_all", On: strings.TrimSpace(m.selectedProject) == ""},
	}
	for _, project := range m.projects {
		items = append(items, chronicleFilterItem{
			Group: "Scope",
			Label: chronicleScopeLabel(project),
			Kind:  "scope_project",
			Value: project,
			On:    m.selectedProject == project,
		})
	}
	for _, kind := range []event.EntryKind{event.RecordKind, event.DecisionKind, event.CommitKind} {
		items = append(items, chronicleFilterItem{
			Group: "Kinds",
			Label: chronicleKindLabel(kind),
			Kind:  "kind",
			Value: string(kind),
			On:    m.kindFilter[kind],
		})
	}
	for _, tag := range m.availableTags {
		items = append(items, chronicleFilterItem{
			Group: "Tags",
			Label: "#" + tag,
			Kind:  "tag",
			Value: tag,
			On:    m.tagFilter[tag],
		})
	}
	return items
}

func (m chronicleModel) timelineWidth() int {
	switch {
	case m.isCompact():
		return m.width - 2
	case m.isMedium():
		leftWidth := min(32, max(30, m.width/4))
		return max(40, m.width-leftWidth-2)
	default:
		leftWidth := min(36, max(32, m.width/4))
		rightWidth := min(44, max(34, m.width/3))
		return max(34, m.width-leftWidth-rightWidth-4)
	}
}

func (m chronicleModel) timelineHeight() int {
	switch {
	case m.isCompact():
		return max(10, m.height-8)
	case m.isMedium():
		return max(10, ((m.height-8)*3)/5)
	default:
		return max(10, m.height-8)
	}
}

func (m chronicleModel) isCompact() bool {
	return m.width < 92
}

func (m chronicleModel) isMedium() bool {
	return m.width >= 92 && m.width < 124
}

func (m chronicleModel) filteredEvents() []event.Event {
	return filterChronicleEvents(m.events, m.filters())
}

func (m chronicleModel) filteredCount() int {
	return len(m.filteredEvents())
}

func (m chronicleModel) activeKindLabels() []string {
	var kinds []string
	for _, kind := range []event.EntryKind{event.RecordKind, event.DecisionKind, event.CommitKind} {
		if m.kindFilter[kind] {
			kinds = append(kinds, chronicleKindLabel(kind))
		}
	}
	return kinds
}

func (m chronicleModel) activeTagLabels() []string {
	tags := tagKeys(m.tagFilter)
	for i := range tags {
		tags[i] = "#" + tags[i]
	}
	return tags
}

func (m chronicleModel) scopeStatusMessage() string {
	return fmt.Sprintf("%s in %s", chronicleCountLabel(m.filteredCount()), chronicleScopeLabel(m.selectedProject))
}

func (m chronicleModel) activeModeLabel() string {
	if m.isCompact() && m.showPreview {
		return "Inspect"
	}
	return "Browse"
}

func (m *chronicleModel) setStatus(status string, tone chronicleStatusTone) {
	m.status = status
	m.statusTone = tone
}

func (m *chronicleModel) setStatusInfo(status string) {
	m.setStatus(status, chronicleStatusInfo)
}

func (m *chronicleModel) setStatusSuccess(status string) {
	m.setStatus(status, chronicleStatusSuccess)
}

func (m *chronicleModel) setStatusWarn(status string) {
	m.setStatus(status, chronicleStatusWarn)
}

func (m *chronicleModel) setStatusError(status string) {
	m.setStatus(status, chronicleStatusError)
}

func loadChronicleDataCmd() tea.Cmd {
	return loadChronicleDataCmdWithHighlight(0)
}

func loadChronicleDataCmdWithHighlight(highlight int64) tea.Cmd {
	return func() tea.Msg {
		s, err := openGlobalStore()
		if err != nil {
			return chronicleDataLoadedMsg{err: err}
		}
		events, err := s.List()
		if err != nil {
			return chronicleDataLoadedMsg{err: err}
		}
		configured, err := getConfiguredTags()
		if err != nil {
			return chronicleDataLoadedMsg{err: err}
		}
		return chronicleDataLoadedMsg{
			events:    events,
			tags:      chronicleUnionTags(configured, events),
			highlight: highlight,
		}
	}
}

func chronicleKindLabel(kind event.EntryKind) string {
	switch kind {
	case event.DecisionKind:
		return "decision"
	case event.CommitKind:
		return "commit"
	default:
		return "record"
	}
}

func chronicleSearchStatus(query string) string {
	if strings.TrimSpace(query) == "" {
		return "Search cleared"
	}
	return fmt.Sprintf("Search applied: %q", strings.TrimSpace(query))
}

func chronicleSearchDraftStatus(query string) string {
	if strings.TrimSpace(query) == "" {
		return "Searching all Chronicle entries"
	}
	return fmt.Sprintf("Searching: %q", strings.TrimSpace(query))
}

func tagKeys(set map[string]bool) []string {
	var tags []string
	for tag, on := range set {
		if on {
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	return tags
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	rs := []rune(s)
	rs[0] = unicode.ToUpper(rs[0])
	for i := 1; i < len(rs); i++ {
		rs[i] = unicode.ToLower(rs[i])
	}
	return string(rs)
}

func truncateLine(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if width == 1 {
		return ansi.Truncate(s, width, "")
	}
	return ansi.Truncate(s, width, "…")
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
