package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/divijg19/sage/internal/event"
)

type chronicleRowKind string

const (
	chronicleRowDay   chronicleRowKind = "day"
	chronicleRowEntry chronicleRowKind = "entry"
)

type chronicleFilters struct {
	Query           string
	Project         string
	EnabledKinds    map[event.EntryKind]bool
	EnabledTags     map[string]bool
	InitialAllScope bool
}

type chronicleRow struct {
	Kind        chronicleRowKind
	DayKey      string
	DayLabel    string
	DayCount    int
	DayOpen     bool
	Event       event.Event
	EntryOpen   bool
	PreviewBody string
}

func filterChronicleEvents(events []event.Event, filters chronicleFilters) []event.Event {
	out := make([]event.Event, 0, len(events))
	query := strings.ToLower(strings.TrimSpace(filters.Query))

	for _, e := range events {
		if strings.TrimSpace(filters.Project) != "" && e.Project != filters.Project {
			continue
		}

		if len(filters.EnabledKinds) > 0 && !filters.EnabledKinds[e.Kind] {
			continue
		}

		if len(filters.EnabledTags) > 0 {
			matched := false
			for _, tag := range e.Tags {
				if filters.EnabledTags[strings.ToLower(strings.TrimSpace(tag))] {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		if query != "" && !chronicleMatchesQuery(e, query) {
			continue
		}

		out = append(out, e)
	}

	return out
}

func chronicleMatchesQuery(e event.Event, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		e.Title,
		e.Content,
		strings.Join(e.Tags, " "),
		e.Project,
		string(e.Kind),
	}, "\n"))
	return strings.Contains(haystack, query)
}

func buildChronicleRows(events []event.Event, collapsedDays map[string]bool, expandedEntries map[int64]bool) []chronicleRow {
	var rows []chronicleRow
	currentDay := ""
	dayCount := 0
	dayStart := 0

	for _, e := range events {
		dayKey := e.Timestamp.Format("2006-01-02")
		if dayKey != currentDay {
			if currentDay != "" {
				rows[dayStart].DayCount = dayCount
			}
			currentDay = dayKey
			dayCount = 0
			dayStart = len(rows)
			rows = append(rows, chronicleRow{
				Kind:     chronicleRowDay,
				DayKey:   dayKey,
				DayLabel: e.Timestamp.Format("Mon, Jan 02 2006"),
				DayOpen:  !collapsedDays[dayKey],
			})
		}

		dayCount++
		if collapsedDays[dayKey] {
			continue
		}

		rows = append(rows, chronicleRow{
			Kind:        chronicleRowEntry,
			DayKey:      dayKey,
			Event:       e,
			EntryOpen:   expandedEntries[e.Seq],
			PreviewBody: chronicleInlinePreview(e.Content),
		})
	}

	if currentDay != "" {
		rows[dayStart].DayCount = dayCount
	}

	return rows
}

func chronicleInlinePreview(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	lines := strings.Split(body, "\n")
	if len(lines) > 4 {
		lines = lines[:4]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func chronicleUnionTags(configured []string, events []event.Event) []string {
	seen := make(map[string]struct{})
	var out []string

	for _, tag := range parseTags(configured) {
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}

	for _, e := range events {
		for _, tag := range parseTags(e.Tags) {
			if _, ok := seen[tag]; ok {
				continue
			}
			seen[tag] = struct{}{}
			out = append(out, tag)
		}
	}

	return out
}

func chronicleProjectOptions(events []event.Event) []string {
	seen := map[string]struct{}{}
	var projects []string
	for _, e := range events {
		project := strings.TrimSpace(e.Project)
		if project == "" {
			project = defaultProjectName
		}
		if _, ok := seen[project]; ok {
			continue
		}
		seen[project] = struct{}{}
		projects = append(projects, project)
	}
	sort.Strings(projects)
	return projects
}

func chronicleScopeLabel(project string) string {
	project = strings.TrimSpace(project)
	if project == "" {
		return "all projects"
	}
	if project == defaultProjectName {
		return "global"
	}
	return project
}

func chronicleCountLabel(n int) string {
	if n == 1 {
		return "1 entry"
	}
	return fmt.Sprintf("%d entries", n)
}

func chroniclePreviewTitle(e *event.Event) string {
	if e == nil {
		return "No entry selected"
	}
	title := strings.TrimSpace(e.Title)
	if title == "" {
		return "(untitled)"
	}
	return title
}

func chronicleBodyExcerpt(body string, maxLines int) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "(no notes yet)"
	}
	lines := strings.Split(body, "\n")
	if maxLines > 0 && len(lines) > maxLines {
		lines = append(lines[:maxLines], "...")
	}
	return strings.Join(lines, "\n")
}

func chronicleDaySummary(day string, count int) string {
	parsed, err := time.Parse("2006-01-02", day)
	if err != nil {
		return chronicleCountLabel(count)
	}
	return fmt.Sprintf("%s · %s", parsed.Format("Monday, January 02"), chronicleCountLabel(count))
}
