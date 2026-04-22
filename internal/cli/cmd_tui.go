package cli

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the Chronicle terminal interface",
	Long: "Open Chronicle, Sage's keyboard-first terminal interface for browsing\n" +
		"and adding entries from the same local event store used by the CLI.",
	RunE: func(cmd *cobra.Command, args []string) error {
		project, filter := resolveProjectFilter(tuiProject, tuiAll)
		if !filter {
			project = ""
		}

		model := newChronicleModel(chronicleOptions{
			Query:   strings.TrimSpace(tuiQuery),
			Project: project,
			Tags:    parseTags(tuiTags),
		})

		program := tea.NewProgram(model, tea.WithAltScreen())
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

type chronicleDataLoadedMsg struct {
	events    []event.Event
	tags      []string
	highlight int64
	err       error
}

type chronicleEditorFinishedMsg struct {
	err error
}

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
		m.queryInput.Width = max(18, min(32, m.width-12))
		return m, nil

	case chronicleDataLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = msg.err.Error()
			return m, nil
		}
		m.events = msg.events
		m.availableTags = msg.tags
		m.projects = chronicleProjectOptions(msg.events)
		m.status = fmt.Sprintf("%s in %s", chronicleCountLabel(len(filterChronicleEvents(m.events, m.filters()))), chronicleScopeLabel(m.selectedProject))
		m.rebuildRows(msg.highlight)
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
			m.status = "Search Chronicle"
			return m, m.queryInput.Focus()
		case "f":
			m.showFilters = true
			m.filterIndex = 0
			m.status = "Filter palette"
		case "n":
			m.openQuickEntry()
			return m, m.titleInput.Focus()
		case "r":
			m.loading = true
			m.status = "Reloading Chronicle..."
			return m, loadChronicleDataCmd()
		case "tab":
			if m.isCompact() {
				m.showPreview = !m.showPreview
			}
		}

		return m, nil
	}

	return m, nil
}

func (m chronicleModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading Chronicle..."
	}

	bodyHeight := max(8, m.height-4)
	header := m.renderHeader()
	status := m.renderStatus()

	var body string
	switch {
	case m.isCompact():
		body = m.renderCompactBody(bodyHeight)
	case m.isMedium():
		body = m.renderMediumBody(bodyHeight)
	default:
		body = m.renderWideBody(bodyHeight)
	}

	if m.showFilters {
		body = m.placeOverlay(body, m.renderFilterOverlay())
	}
	if m.showQuick {
		body = m.placeOverlay(body, m.renderQuickEntryOverlay())
	}
	if m.showPreview && m.isCompact() {
		body = m.placeOverlay(body, m.renderPreviewOverlay())
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, status)
}

func (m chronicleModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.focused = ""
		m.queryInput.Blur()
		m.query = m.queryInput.Value()
		m.rebuildRows(0)
		m.status = fmt.Sprintf("Search cleared to %q", strings.TrimSpace(m.query))
		return m, nil
	case "enter":
		m.focused = ""
		m.queryInput.Blur()
		m.query = m.queryInput.Value()
		m.rebuildRows(0)
		return m, nil
	}

	var cmd tea.Cmd
	m.queryInput, cmd = m.queryInput.Update(msg)
	m.query = m.queryInput.Value()
	m.rebuildRows(0)
	return m, cmd
}

func (m chronicleModel) updateQuickEntry(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeQuickEntry("Quick entry canceled")
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

	switch msg.String() {
	case "esc":
		m.showFilters = false
		m.status = "Closed filters"
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
		case "scope_project":
			m.selectedProject = item.Value
		case "kind":
			kind := event.EntryKind(item.Value)
			m.kindFilter[kind] = !m.kindFilter[kind]
			if !m.hasEnabledKinds() {
				m.kindFilter[kind] = true
			}
		case "tag":
			if m.tagFilter[item.Value] {
				delete(m.tagFilter, item.Value)
			} else {
				m.tagFilter[item.Value] = true
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
			m.status = execErr.Error()
			return m, nil
		}
	} else {
		content, err := launch.result()
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		edited = content
	}

	s, err := openGlobalStore()
	if err != nil {
		m.status = err.Error()
		return m, nil
	}

	req.Edited = edited
	result, err := entryflow.Finalize(req, entryflow.Dependencies{
		Store:       s,
		EnsureTags:  ensureTagsConfigured,
		ResolveKind: resolveKind,
	})
	if err != nil {
		m.status = err.Error()
		return m, nil
	}

	switch result.Status {
	case entryflow.StatusSaved:
		m.status = "Entry recorded"
		m.showQuick = false
		m.focused = ""
		return m, loadChronicleDataCmdWithHighlight(result.Event.Seq)
	case entryflow.StatusCanceled:
		m.status = "Editor canceled"
	case entryflow.StatusUnchanged:
		m.status = "No changes recorded"
	case entryflow.StatusEmpty:
		m.status = "Entry was empty"
	case entryflow.StatusDuplicate:
		m.status = "Duplicate entry skipped"
	}

	m.showQuick = false
	m.focused = ""
	return m, loadChronicleDataCmd()
}

func (m *chronicleModel) startQuickEntry() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(m.titleInput.Value())
	if title == "" {
		m.status = "Quick entry needs a title"
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
		m.status = err.Error()
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
	m.status = "Opening editor..."
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
}

func (m *chronicleModel) closeQuickEntry(status string) {
	m.showQuick = false
	m.focused = ""
	m.titleInput.Blur()
	m.tagsInput.Blur()
	m.status = status
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
	m.status = "Quick entry kind: " + chronicleKindLabel(m.quickKind)
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
	case chronicleRowEntry:
		m.expandedEntries[row.Event.Seq] = !m.expandedEntries[row.Event.Seq]
		m.rebuildRows(row.Event.Seq)
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
		{Label: "All projects", Kind: "scope_all", On: strings.TrimSpace(m.selectedProject) == ""},
	}
	for _, project := range m.projects {
		items = append(items, chronicleFilterItem{
			Label: chronicleScopeLabel(project),
			Kind:  "scope_project",
			Value: project,
			On:    m.selectedProject == project,
		})
	}
	for _, kind := range []event.EntryKind{event.RecordKind, event.DecisionKind, event.CommitKind} {
		items = append(items, chronicleFilterItem{
			Label: chronicleKindLabel(kind),
			Kind:  "kind",
			Value: string(kind),
			On:    m.kindFilter[kind],
		})
	}
	for _, tag := range m.availableTags {
		items = append(items, chronicleFilterItem{
			Label: "#" + tag,
			Kind:  "tag",
			Value: tag,
			On:    m.tagFilter[tag],
		})
	}
	return items
}

func (m chronicleModel) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("25")).
		Padding(0, 1).
		Render("Chronicle")

	scope := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Render(chronicleScopeLabel(m.selectedProject))

	count := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(chronicleCountLabel(len(filterChronicleEvents(m.events, m.filters()))))

	hints := []string{"j/k move", "enter expand", "/ search", "f filters", "n quick entry", "r reload", "q quit"}
	if m.isCompact() {
		hints = append(hints, "tab preview")
	}
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Render(strings.Join(hints, " · "))

	line := lipgloss.JoinHorizontal(lipgloss.Left, title, " ", scope, "  ", count)
	return lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		MarginBottom(1).
		Render(lipgloss.JoinVertical(lipgloss.Left, line, help))
}

func (m chronicleModel) renderStatus() string {
	return lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		Foreground(lipgloss.Color("244")).
		MarginTop(1).
		Render(m.status)
}

func (m chronicleModel) renderWideBody(height int) string {
	leftWidth := 28
	rightWidth := min(42, max(32, m.width/3))
	centerWidth := max(30, m.width-leftWidth-rightWidth-4)

	left := m.renderRail(leftWidth, height)
	center := m.renderTimeline(centerWidth, height)
	right := m.renderPreview(rightWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", center, "  ", right)
}

func (m chronicleModel) renderMediumBody(height int) string {
	leftWidth := 26
	centerWidth := max(36, m.width-leftWidth-3)
	timelineHeight := max(8, (height*3)/5)
	previewHeight := max(6, height-timelineHeight-1)

	left := m.renderRail(leftWidth, height)
	center := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderTimeline(centerWidth, timelineHeight),
		m.renderPreview(centerWidth, previewHeight),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", center)
}

func (m chronicleModel) renderCompactBody(height int) string {
	return m.renderTimeline(m.width-2, height)
}

func (m chronicleModel) renderRail(width int, height int) string {
	box := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(1)

	kindBits := []string{}
	for _, kind := range []event.EntryKind{event.RecordKind, event.DecisionKind, event.CommitKind} {
		if m.kindFilter[kind] {
			kindBits = append(kindBits, chronicleKindLabel(kind))
		}
	}
	if len(kindBits) == 0 {
		kindBits = append(kindBits, "none")
	}

	tagBits := []string{"(none)"}
	if len(m.tagFilter) > 0 {
		tagBits = tagKeys(m.tagFilter)
		for i := range tagBits {
			tagBits[i] = "#" + tagBits[i]
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Search"),
		m.renderSearchBox(width-4),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Scope"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(chronicleScopeLabel(m.selectedProject)),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Kinds"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(strings.Join(kindBits, " · ")),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Tags"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(strings.Join(tagBits, " ")),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Press f to adjust filters"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Press n to start a quick entry"),
	)

	return box.Render(content)
}

func (m chronicleModel) renderSearchBox(width int) string {
	input := m.queryInput
	input.Width = max(12, width-2)
	borderColor := lipgloss.Color("238")
	if m.focused == "search" {
		borderColor = lipgloss.Color("39")
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(input.View())
}

func (m chronicleModel) renderTimeline(width int, height int) string {
	box := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(1)

	if len(m.rows) == 0 {
		return box.Render(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("No Chronicle entries match the current filters."))
	}

	contentWidth := max(20, width-4)
	var allLines []string
	for i, row := range m.rows {
		allLines = append(allLines, m.renderTimelineRow(row, i == m.selectedRow, contentWidth)...)
	}

	viewHeight := max(1, height-2)
	if len(allLines) <= viewHeight {
		return box.Render(strings.Join(allLines, "\n"))
	}

	start := min(m.scrollLine, max(0, len(allLines)-viewHeight))
	end := min(len(allLines), start+viewHeight)
	return box.Render(strings.Join(allLines[start:end], "\n"))
}

func (m chronicleModel) renderTimelineRow(row chronicleRow, selected bool, width int) []string {
	switch row.Kind {
	case chronicleRowDay:
		prefix := "▾"
		if !row.DayOpen {
			prefix = "▸"
		}
		line := fmt.Sprintf("%s %s  %s", prefix, row.DayLabel, chronicleCountLabel(row.DayCount))
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Bold(true)
		if selected {
			style = style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24"))
		}
		return []string{truncateLine(style.Render(line), width)}
	default:
		kind := chronicleKindLabel(row.Event.Kind)
		title := strings.TrimSpace(row.Event.Title)
		if title == "" {
			title = "(untitled)"
		}
		tags := ""
		if len(row.Event.Tags) > 0 {
			tagParts := append([]string(nil), row.Event.Tags...)
			sort.Strings(tagParts)
			for i := range tagParts {
				tagParts[i] = "#" + tagParts[i]
			}
			tags = " " + strings.Join(tagParts, " ")
		}

		line := fmt.Sprintf("│ [%d] %s %-8s %s%s", row.Event.Seq, row.Event.Timestamp.Format("15:04"), kind, title, tags)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		if selected {
			style = style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24"))
		}
		lines := []string{truncateLine(style.Render(line), width)}
		if row.EntryOpen {
			excerpt := chronicleBodyExcerpt(row.PreviewBody, 4)
			for _, detail := range strings.Split(excerpt, "\n") {
				detailLine := lipgloss.NewStyle().
					Foreground(lipgloss.Color("245")).
					Render("│    " + detail)
				if selected {
					detailLine = lipgloss.NewStyle().
						Foreground(lipgloss.Color("252")).
						Background(lipgloss.Color("24")).
						Render("│    " + detail)
				}
				lines = append(lines, truncateLine(detailLine, width))
			}
		}
		return lines
	}
}

func (m chronicleModel) renderPreview(width int, height int) string {
	box := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(1)

	row := m.selected()
	if row == nil {
		return box.Render("No entry selected")
	}
	if row.Kind == chronicleRowDay {
		return box.Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render(row.DayLabel),
			lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(chronicleDaySummary(row.DayKey, row.DayCount)),
		))
	}

	e := row.Event
	project := chronicleScopeLabel(e.Project)
	if strings.TrimSpace(e.Project) == "" {
		project = "global"
	}
	tags := "(none)"
	if len(e.Tags) > 0 {
		tagParts := append([]string(nil), e.Tags...)
		sort.Strings(tagParts)
		for i := range tagParts {
			tagParts[i] = "#" + tagParts[i]
		}
		tags = strings.Join(tagParts, " ")
	}

	bodyLines := max(4, height-11)
	content := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render(chroniclePreviewTitle(&e)),
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(fmt.Sprintf("[%d] %s · %s", e.Seq, e.Timestamp.Format("2006-01-02 15:04"), chronicleKindLabel(e.Kind))),
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Project: "+project),
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Tags: "+tags),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(chronicleBodyExcerpt(e.Content, bodyLines)),
	)
	return box.Render(content)
}

func (m chronicleModel) renderFilterOverlay() string {
	items := m.filterItems()
	width := min(56, max(42, m.width-8))
	height := min(max(10, len(items)+4), max(12, m.height-6))
	box := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(1)

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Filter Chronicle"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Space toggles · esc closes"),
	}
	visible := height - 4
	start := 0
	if m.filterIndex >= visible {
		start = m.filterIndex - visible + 1
	}
	end := min(len(items), start+visible)
	for i := start; i < end; i++ {
		item := items[i]
		cursor := " "
		if i == m.filterIndex {
			cursor = "›"
		}
		check := "○"
		if item.On {
			check = "●"
		}
		line := fmt.Sprintf("%s %s %s", cursor, check, item.Label)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		if i == m.filterIndex {
			style = style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24"))
		}
		lines = append(lines, style.Render(line))
	}
	return box.Render(strings.Join(lines, "\n"))
}

func (m chronicleModel) renderQuickEntryOverlay() string {
	width := min(64, max(42, m.width-10))
	box := lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(1)

	titleBorder := lipgloss.Color("238")
	tagsBorder := lipgloss.Color("238")
	kindBorder := lipgloss.Color("238")
	if m.quickField == 0 {
		titleBorder = lipgloss.Color("39")
	}
	if m.quickField == 1 {
		tagsBorder = lipgloss.Color("39")
	}
	if m.quickField == 2 {
		kindBorder = lipgloss.Color("39")
	}

	titleBox := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(titleBorder).Padding(0, 1).Render(m.titleInput.View())
	tagsBox := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(tagsBorder).Padding(0, 1).Render(m.tagsInput.View())
	kindBox := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(kindBorder).Padding(0, 1).Render(strings.Title(m.quickKindLabel()))

	return box.Render(lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Render("Quick Entry"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Seed the note here, then Chronicle opens your editor."),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Title"),
		titleBox,
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Tags"),
		tagsBox,
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Kind"),
		kindBox,
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Tab moves · left/right toggles kind · enter continues · esc cancels"),
	))
}

func (m chronicleModel) renderPreviewOverlay() string {
	width := min(64, max(42, m.width-8))
	return lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(1).
		Render(m.renderPreview(width-4, min(18, max(10, m.height-8))))
}

func (m chronicleModel) placeOverlay(base string, overlay string) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	if len(baseLines) == 0 {
		return overlay
	}

	startY := max(0, (len(baseLines)-len(overlayLines))/2)
	for i, line := range overlayLines {
		target := startY + i
		if target >= len(baseLines) {
			break
		}
		baseLine := []rune(baseLines[target])
		overlayRunes := []rune(line)
		startX := max(0, (m.width-len(overlayRunes))/2)
		if len(baseLine) < m.width {
			baseLine = append(baseLine, []rune(strings.Repeat(" ", m.width-len(baseLine)))...)
		}
		for j, r := range overlayRunes {
			if startX+j >= len(baseLine) {
				break
			}
			if r != ' ' {
				baseLine[startX+j] = r
			}
		}
		baseLines[target] = string(baseLine)
	}
	return strings.Join(baseLines, "\n")
}

func (m chronicleModel) timelineWidth() int {
	switch {
	case m.isCompact():
		return m.width - 2
	case m.isMedium():
		return max(36, m.width-29)
	default:
		rightWidth := min(42, max(32, m.width/3))
		return max(30, m.width-28-rightWidth-4)
	}
}

func (m chronicleModel) timelineHeight() int {
	switch {
	case m.isCompact():
		return max(8, m.height-4)
	case m.isMedium():
		return max(8, ((m.height-4)*3)/5)
	default:
		return max(8, m.height-4)
	}
}

func (m chronicleModel) isCompact() bool {
	return m.width < 90
}

func (m chronicleModel) isMedium() bool {
	return m.width >= 90 && m.width < 120
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

func truncateLine(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
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
