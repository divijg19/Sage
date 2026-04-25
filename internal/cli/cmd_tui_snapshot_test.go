package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestChronicleSnapshots(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)

	cases := map[string]chronicleModel{
		"wide_default":    fixtureChronicleModel(140, 34),
		"medium_default":  fixtureChronicleModel(108, 32),
		"compact_browse":  fixtureChronicleModel(80, 24),
		"compact_inspect": fixtureCompactInspectModel(),
		"filters_overlay": fixtureFiltersOverlayModel(),
		"search_active":   fixtureSearchActiveModel(),
		"empty_results":   fixtureEmptyResultsModel(),
		"loading_state":   fixtureLoadingModel(),
		"quick_entry":     fixtureQuickEntryModel(),
		"error_state":     fixtureErrorModel(),
	}

	for name, model := range cases {
		t.Run(name, func(t *testing.T) {
			raw := model.View()
			if !strings.Contains(raw, "\x1b[") {
				t.Fatalf("expected ANSI styling in raw view for %s", name)
			}
			assertChronicleWidthSafe(t, raw, model.width)

			stripped := ansi.Strip(raw)
			goldenPath := filepath.Join("testdata", "chronicle_"+name+".golden")
			expected, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden %s: %v\nactual:\n%s", goldenPath, err, normalizeChronicleSnapshot(stripped))
			}

			actual := normalizeChronicleSnapshot(stripped)
			want := normalizeChronicleSnapshot(string(expected))
			if actual != want {
				t.Fatalf("snapshot mismatch for %s\nexpected:\n%s\nactual:\n%s", name, want, actual)
			}
		})
	}
}

func fixtureCompactInspectModel() chronicleModel {
	m := fixtureChronicleModel(80, 24)
	m.showPreview = true
	m.setStatusInfo("Inspector mode")
	return m
}

func fixtureFiltersOverlayModel() chronicleModel {
	m := fixtureChronicleModel(140, 34)
	m.showFilters = true
	m.filterIndex = 4
	m.setStatusInfo("Filter Chronicle")
	return m
}

func fixtureSearchActiveModel() chronicleModel {
	m := fixtureChronicleModel(140, 34)
	m.focused = "search"
	m.query = "auth"
	m.queryInput.SetValue("auth")
	m.rebuildRows(0)
	m.setStatusInfo(`Searching: "auth"`)
	return m
}

func fixtureEmptyResultsModel() chronicleModel {
	m := fixtureChronicleModel(140, 34)
	m.query = "missing phrase"
	m.queryInput.SetValue("missing phrase")
	m.rebuildRows(0)
	m.setStatusInfo(`Search applied: "missing phrase"`)
	return m
}

func fixtureLoadingModel() chronicleModel {
	m := newChronicleModel(chronicleOptions{})
	m.width = 140
	m.height = 34
	return m
}

func fixtureQuickEntryModel() chronicleModel {
	m := fixtureChronicleModel(140, 34)
	m.openQuickEntry()
	m.titleInput.SetValue("Polish Chronicle")
	m.tagsInput.SetValue("ux,release")
	return m
}

func fixtureErrorModel() chronicleModel {
	m := fixtureChronicleModel(140, 34)
	m.setStatusError("Could not open Chronicle store")
	return m
}

func assertChronicleWidthSafe(t *testing.T, raw string, width int) {
	t.Helper()
	for i, line := range strings.Split(raw, "\n") {
		if got := ansi.StringWidth(line); got > width {
			t.Fatalf("line %d exceeds visible width %d: got %d", i, width, got)
		}
	}
}

func normalizeChronicleSnapshot(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
		trimmedLeft := strings.TrimLeft(lines[i], " ")
		if strings.HasPrefix(trimmedLeft, "╔") || strings.HasPrefix(trimmedLeft, "║") || strings.HasPrefix(trimmedLeft, "╚") {
			lines[i] = trimmedLeft
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}
