package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/divijg19/sage/internal/event"
)

func (m chronicleModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading Chronicle..."
	}

	theme := newChronicleTheme()
	header := m.renderMasthead(theme)
	status := m.renderStatus(theme)
	bodyHeight := max(10, m.height-lipgloss.Height(header)-lipgloss.Height(status))

	var body string
	switch {
	case m.isCompact():
		body = m.renderCompactBody(theme, bodyHeight)
	case m.isMedium():
		body = m.renderMediumBody(theme, bodyHeight)
	default:
		body = m.renderWideBody(theme, bodyHeight)
	}

	if m.showFilters {
		body = m.placeOverlay(body, m.renderFilterOverlay(theme))
	}
	if m.showQuick {
		body = m.placeOverlay(body, m.renderQuickEntryOverlay(theme))
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, status)
}

func (m chronicleModel) renderMasthead(theme chronicleTheme) string {
	contentWidth := max(20, m.width-4)

	topTokens := []string{
		theme.badge("Chronicle"),
		theme.chip(chronicleScopeLabel(m.selectedProject), true, true),
		theme.chip(chronicleCountLabel(m.filteredCount()), true, false),
	}
	if m.isCompact() {
		topTokens = append(topTokens, theme.chip(m.activeModeLabel(), true, false))
	}

	searchSummary := "Search: all notes"
	if strings.TrimSpace(m.query) != "" {
		searchSummary = fmt.Sprintf("Search: %q", strings.TrimSpace(m.query))
	}
	filterSummary := chronicleFilterSummary(m)
	help := chronicleHelpSummary(m)

	lines := []string{
		truncateLine(strings.Join(topTokens, " "), contentWidth),
		truncateLine(theme.muted().Render(searchSummary)+"  "+theme.muted().Render(filterSummary), contentWidth),
		truncateLine(theme.muted().Render(help), contentWidth),
	}

	return theme.masthead(m.width).Render(strings.Join(lines, "\n"))
}

func (m chronicleModel) renderStatus(theme chronicleTheme) string {
	return theme.status(m.statusTone, m.width).Render(m.status)
}

func (m chronicleModel) renderWideBody(theme chronicleTheme, height int) string {
	gapWidth := 4
	leftWidth := min(36, max(32, m.width/4))
	rightWidth := min(44, max(34, m.width/3))
	centerWidth := max(34, m.width-leftWidth-rightWidth-gapWidth)

	left := m.renderRail(theme, leftWidth, height)
	center := m.renderTimeline(theme, centerWidth, height)
	right := m.renderPreview(theme, rightWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", center, "  ", right)
}

func (m chronicleModel) renderMediumBody(theme chronicleTheme, height int) string {
	gapWidth := 2
	leftWidth := min(32, max(30, m.width/4))
	centerWidth := max(40, m.width-leftWidth-gapWidth)
	timelineHeight := max(10, (height*3)/5)
	previewHeight := max(9, height-timelineHeight-1)

	left := m.renderRail(theme, leftWidth, height)
	center := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderTimeline(theme, centerWidth, timelineHeight),
		m.renderPreview(theme, centerWidth, previewHeight),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", center)
}

func (m chronicleModel) renderCompactBody(theme chronicleTheme, height int) string {
	width := m.width - 2
	if m.showPreview {
		return m.renderPreview(theme, width, height)
	}
	return m.renderTimeline(theme, width, height)
}

func (m chronicleModel) renderRail(theme chronicleTheme, width int, height int) string {
	contentWidth := max(18, width-4)

	activeKinds := m.activeKindLabels()
	kindTokens := make([]string, 0, len(activeKinds))
	for _, kind := range activeKinds {
		kindTokens = append(kindTokens, theme.chip(titleCase(kind), true, false))
	}
	if len(kindTokens) == 0 {
		kindTokens = append(kindTokens, theme.chip("None", false, false))
	}

	activeTags := m.activeTagLabels()
	tagTokens := make([]string, 0, len(activeTags))
	for _, tag := range activeTags {
		tagTokens = append(tagTokens, theme.chip(tag, true, false))
	}
	if len(tagTokens) == 0 {
		tagTokens = append(tagTokens, theme.chip("No tag filters", false, false))
	}

	actionTokens := []string{
		theme.chip("/ search", true, false),
		theme.chip("f filters", true, false),
		theme.chip("n quick entry", true, false),
		theme.chip("r reload", true, false),
	}

	sections := []string{
		theme.sectionTitle().Render("Context"),
		m.renderSearchBox(theme, contentWidth),
		"",
		theme.sectionTitle().Render("Scope"),
		theme.body().Render(chronicleScopeLabel(m.selectedProject)),
		"",
		theme.sectionTitle().Render("Kinds"),
		wrapStyledTokens(kindTokens, contentWidth),
		"",
		theme.sectionTitle().Render("Tags"),
		wrapStyledTokens(tagTokens, contentWidth),
		"",
		theme.sectionTitle().Render("Actions"),
		wrapStyledTokens(actionTokens, contentWidth),
	}

	return theme.panel(width, height, false).Render(strings.Join(sections, "\n"))
}

func (m chronicleModel) renderSearchBox(theme chronicleTheme, width int) string {
	input := m.queryInput
	boxWidth := max(14, width-4)
	input.Width = max(10, width-8)
	return theme.inputBox(m.focused == "search").Width(boxWidth).Render(input.View())
}

func (m chronicleModel) renderTimeline(theme chronicleTheme, width int, height int) string {
	contentWidth := max(22, width-4)
	heading := truncateLine(theme.sectionTitle().Render("Timeline")+"  "+theme.muted().Render(m.scopeStatusMessage()), contentWidth)

	if len(m.rows) == 0 {
		body := lipgloss.JoinVertical(
			lipgloss.Left,
			heading,
			"",
			theme.body().Render("No Chronicle entries match the current filters."),
			theme.muted().Render("Adjust search or filters to broaden the view."),
		)
		return theme.panel(width, height, false).Render(body)
	}

	var allLines []string
	for i, row := range m.rows {
		allLines = append(allLines, m.renderTimelineRow(row, i == m.selectedRow, contentWidth)...)
	}

	viewHeight := max(1, height-6)
	start := min(m.scrollLine, max(0, len(allLines)-viewHeight))
	end := min(len(allLines), start+viewHeight)
	bodyLines := allLines[start:end]

	body := lipgloss.JoinVertical(lipgloss.Left, append([]string{heading, ""}, bodyLines...)...)
	return theme.panel(width, height, false).Render(body)
}

func (m chronicleModel) renderTimelineRow(row chronicleRow, selected bool, width int) []string {
	theme := newChronicleTheme()
	selectedStyle := theme.selectedRow()

	switch row.Kind {
	case chronicleRowDay:
		prefix := "▾"
		if !row.DayOpen {
			prefix = "▸"
		}
		line := fmt.Sprintf("%s %s  %s", prefix, row.DayLabel, chronicleCountLabel(row.DayCount))
		style := theme.sectionTitle()
		if selected {
			style = selectedStyle.Copy().Bold(true)
		}
		return []string{truncateLine(style.Render(line), width)}
	default:
		title := strings.TrimSpace(row.Event.Title)
		if title == "" {
			title = "(untitled)"
		}

		project := chronicleScopeLabel(row.Event.Project)
		metaParts := []string{titleCase(chronicleKindLabel(row.Event.Kind)), fmt.Sprintf("ID %d", row.Event.Seq), project}
		if len(row.Event.Tags) > 0 {
			tagParts := append([]string(nil), row.Event.Tags...)
			sort.Strings(tagParts)
			for i := range tagParts {
				tagParts[i] = "#" + tagParts[i]
			}
			metaParts = append(metaParts, strings.Join(tagParts, " "))
		}

		marker := " "
		if selected {
			marker = "›"
		}
		head := fmt.Sprintf("%s %s  %s", marker, row.Event.Timestamp.Format("15:04"), title)
		sub := "  " + strings.Join(metaParts, " · ")

		headStyle := theme.body()
		subStyle := theme.muted()
		if selected {
			headStyle = selectedStyle.Copy().Bold(true)
			subStyle = selectedStyle.Copy().Foreground(theme.textSoft)
		}

		lines := []string{
			truncateLine(headStyle.Render(head), width),
			truncateLine(subStyle.Render(sub), width),
		}
		if row.EntryOpen {
			excerpt := chronicleBodyExcerpt(row.PreviewBody, 4)
			for _, detail := range strings.Split(excerpt, "\n") {
				line := "    " + detail
				style := theme.muted()
				if selected {
					style = selectedStyle.Copy().Foreground(theme.textSoft)
				}
				lines = append(lines, truncateLine(style.Render(line), width))
			}
		}
		return lines
	}
}

func (m chronicleModel) renderPreview(theme chronicleTheme, width int, height int) string {
	contentWidth := max(22, width-4)
	lines := []string{truncateLine(theme.sectionTitle().Render("Inspector"), contentWidth), ""}

	row := m.selected()
	if row == nil {
		lines = append(lines,
			theme.body().Render("No entry selected."),
			theme.muted().Render("Move through the timeline to inspect a day or entry."),
		)
		return theme.panel(width, height, false).Render(strings.Join(lines, "\n"))
	}

	if row.Kind == chronicleRowDay {
		lines = append(lines,
			theme.title().Render(row.DayLabel),
			theme.muted().Render(chronicleDaySummary(row.DayKey, row.DayCount)),
			"",
			theme.body().Render("This is a day group."),
			theme.muted().Render("Press Enter to collapse or expand the entries for this day."),
		)
		return theme.panel(width, height, false).Render(strings.Join(lines, "\n"))
	}

	e := m.selectedEvent()
	if e == nil {
		lines = append(lines, theme.body().Render("No entry selected."))
		return theme.panel(width, height, false).Render(strings.Join(lines, "\n"))
	}

	project := chronicleScopeLabel(e.Project)
	if strings.TrimSpace(e.Project) == "" {
		project = "global"
	}

	metaTokens := []string{
		theme.chip(titleCase(chronicleKindLabel(e.Kind)), true, true),
		theme.chip(fmt.Sprintf("[%d]", e.Seq), true, false),
		theme.chip(e.Timestamp.Format("2006-01-02 15:04"), true, false),
		theme.chip("Project: "+project, true, false),
	}

	tagTokens := []string{theme.chip("Tags: (none)", false, false)}
	if len(e.Tags) > 0 {
		tagParts := append([]string(nil), e.Tags...)
		sort.Strings(tagParts)
		tagTokens = tagTokens[:0]
		for _, tag := range tagParts {
			tagTokens = append(tagTokens, theme.chip("#"+tag, true, false))
		}
	}

	bodyLines := max(4, height-16)
	lines = append(lines,
		truncateLine(theme.title().Render(chroniclePreviewTitle(e)), contentWidth),
		wrapStyledTokens(metaTokens, contentWidth),
		"",
		theme.sectionTitle().Render("Tags"),
		wrapStyledTokens(tagTokens, contentWidth),
		"",
		theme.sectionTitle().Render("Notes"),
		theme.body().Render(chronicleBodyExcerpt(e.Content, bodyLines)),
	)

	return theme.panel(width, height, false).Render(strings.Join(lines, "\n"))
}

func (m chronicleModel) renderFilterOverlay(theme chronicleTheme) string {
	items := m.filterItems()
	width := min(62, max(44, m.width-10))
	height := min(max(14, len(items)+8), max(14, m.height-8))
	bodyHeight := max(4, height-8)

	start := 0
	if m.filterIndex >= bodyHeight {
		start = m.filterIndex - bodyHeight + 1
	}
	end := min(len(items), start+bodyHeight)

	lines := []string{
		theme.title().Render("Filter Chronicle"),
		theme.muted().Render("Scope, kinds, and tags"),
		"",
	}

	group := ""
	for i := start; i < end; i++ {
		item := items[i]
		if item.Group != group {
			group = item.Group
			lines = append(lines, theme.sectionTitle().Render(group))
		}
		cursor := " "
		if i == m.filterIndex {
			cursor = "›"
		}
		check := "○"
		if item.On {
			check = "●"
		}
		line := fmt.Sprintf("%s %s %s", cursor, check, item.Label)
		style := theme.body()
		if i == m.filterIndex {
			style = theme.selectedRow()
		}
		lines = append(lines, style.Render(line))
	}

	lines = append(lines, "", theme.muted().Render("Space toggles · Esc closes"))
	return theme.modal(width).Render(strings.Join(lines, "\n"))
}

func (m chronicleModel) renderQuickEntryOverlay(theme chronicleTheme) string {
	width := min(66, max(46, m.width-12))

	titleBox := theme.inputBox(m.quickField == 0).Render(m.titleInput.View())
	tagsBox := theme.inputBox(m.quickField == 1).Render(m.tagsInput.View())
	kindBox := theme.inputBox(m.quickField == 2).Render(titleCase(m.quickKindLabel()))

	lines := []string{
		theme.title().Render("Quick Entry"),
		theme.muted().Render("Seed the note here, then Chronicle opens your editor."),
		"",
		theme.sectionTitle().Render("Title"),
		titleBox,
		"",
		theme.sectionTitle().Render("Tags"),
		tagsBox,
		"",
		theme.sectionTitle().Render("Kind"),
		kindBox,
		"",
		theme.muted().Render("Tab moves · Left/right toggles kind · Enter continues · Esc cancels"),
	}

	return theme.modal(width).Render(strings.Join(lines, "\n"))
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

		overlayLine := line
		overlayWidth := ansi.StringWidth(overlayLine)
		if overlayWidth > m.width {
			overlayLine = ansi.Truncate(overlayLine, m.width, "")
			overlayWidth = ansi.StringWidth(overlayLine)
		}

		startX := max(0, (m.width-overlayWidth)/2)
		baseLine := strings.Repeat(" ", m.width)

		left := ansi.Cut(baseLine, 0, startX)
		rightStart := min(m.width, startX+overlayWidth)
		right := ansi.Cut(baseLine, rightStart, m.width)
		baseLines[target] = left + overlayLine + right
	}
	return strings.Join(baseLines, "\n")
}

func chronicleFilterSummary(m chronicleModel) string {
	parts := []string{}
	if !allKindsEnabled(m.kindFilter) {
		parts = append(parts, "Kinds: "+strings.Join(m.activeKindLabels(), ", "))
	}
	if len(m.tagFilter) > 0 {
		parts = append(parts, "Tags: "+strings.Join(m.activeTagLabels(), " "))
	}
	if len(parts) == 0 {
		return "Filters: none"
	}
	return "Filters: " + strings.Join(parts, " · ")
}

func chronicleHelpSummary(m chronicleModel) string {
	if m.isCompact() {
		return strings.Join([]string{"j/k move", "enter toggle", "/ search", "f filters", "tab inspect", "q quit"}, " · ")
	}
	hints := []string{"j/k move", "enter toggle", "/ search", "f filters", "n quick entry", "r reload"}
	hints = append(hints, "q quit")
	return strings.Join(hints, " · ")
}

func allKindsEnabled(kindFilter map[event.EntryKind]bool) bool {
	return kindFilter[event.RecordKind] && kindFilter[event.DecisionKind] && kindFilter[event.CommitKind]
}

func wrapStyledTokens(tokens []string, width int) string {
	if len(tokens) == 0 {
		return ""
	}
	var lines []string
	current := ""
	currentWidth := 0
	for _, token := range tokens {
		tokenWidth := ansi.StringWidth(token)
		if current == "" {
			current = token
			currentWidth = tokenWidth
			continue
		}
		if currentWidth+1+tokenWidth <= width {
			current += " " + token
			currentWidth += 1 + tokenWidth
			continue
		}
		lines = append(lines, current)
		current = token
		currentWidth = tokenWidth
	}
	if current != "" {
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n")
}
