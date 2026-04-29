package cli

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestTruncateLinePreservesVisibleWidthForANSI(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render("abcdef")
	got := truncateLine(styled, 4)

	if width := ansi.StringWidth(got); width != 4 {
		t.Fatalf("expected visible width 4, got %d (%q)", width, got)
	}
	if plain := ansi.Strip(got); plain != "abc…" {
		t.Fatalf("expected truncated plain text %q, got %q", "abc…", plain)
	}
}

func TestPlaceOverlayCentersANSIContent(t *testing.T) {
	m := chronicleModel{width: 20}
	baseLine := strings.Repeat(".", 20)
	base := strings.Join([]string{baseLine, baseLine, baseLine}, "\n")
	overlay := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render("BOX")

	out := m.placeOverlay(base, overlay)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	center := ansi.Strip(lines[1])
	if got, want := strings.Index(center, "BOX"), 8; got != want {
		t.Fatalf("expected overlay at x=%d, got x=%d (%q)", want, got, center)
	}
	if ansi.Strip(lines[0]) != baseLine || ansi.Strip(lines[2]) != baseLine {
		t.Fatalf("expected non-target lines to remain unchanged")
	}
}

func TestPlaceOverlayTruncatesTooWideOverlay(t *testing.T) {
	m := chronicleModel{width: 6}
	base := strings.Repeat(".", 6)
	overlay := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render("ABCDEFGHI")

	out := m.placeOverlay(base, overlay)
	plain := ansi.Strip(out)
	if got, want := ansi.StringWidth(plain), 6; got != want {
		t.Fatalf("expected visible width %d, got %d (%q)", want, got, plain)
	}
	if plain != "ABCDEF" {
		t.Fatalf("expected overlay to be truncated to base width, got %q", plain)
	}
}

func TestChronicleBodyLinesFillWidth(t *testing.T) {
	for _, width := range []int{80, 92, 108, 124, 140, 180, 220} {
		m := fixtureChronicleModel(width, 34)
		theme := newChronicleTheme()
		height := max(10, m.availableBodyHeight())

		var body string
		switch {
		case m.isCompact():
			body = m.renderCompactBody(theme, height)
		case m.isMedium():
			body = m.renderMediumBody(theme, height)
		default:
			body = m.renderWideBody(theme, height)
		}

		for i, line := range strings.Split(ansi.Strip(body), "\n") {
			if got := ansi.StringWidth(line); got != width {
				t.Fatalf("width %d: expected body line %d to fill width, got %d: %q", width, i, got, line)
			}
		}
	}
}

func TestChronicleViewFillsTerminalHeight(t *testing.T) {
	for _, size := range []struct {
		width  int
		height int
	}{
		{80, 24},
		{108, 32},
		{140, 34},
		{180, 44},
	} {
		m := fixtureChronicleModel(size.width, size.height)
		lines := strings.Split(ansi.Strip(m.View()), "\n")
		if got := len(lines); got != size.height {
			t.Fatalf("expected view %dx%d to fill terminal height, got %d lines", size.width, size.height, got)
		}
	}
}

func TestChronicleFooterWrapsAllShortcuts(t *testing.T) {
	m := fixtureChronicleModel(80, 24)
	footer := ansi.Strip(m.renderFooter(newChronicleTheme()))
	for _, want := range []string{"/", "search", ":", "command", "f", "filters", "n", "new", "r", "reload", "tab", "inspect", "esc", "close", "q", "quit"} {
		if !strings.Contains(footer, want) {
			t.Fatalf("expected compact footer to include %q in:\n%s", want, footer)
		}
	}
	for i, line := range strings.Split(footer, "\n") {
		if got := ansi.StringWidth(line); got > m.width {
			t.Fatalf("footer line %d exceeds width %d: got %d", i, m.width, got)
		}
	}
}
