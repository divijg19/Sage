package cli

import "github.com/charmbracelet/lipgloss"

type chronicleTheme struct {
	surface      lipgloss.Color
	surfaceAlt   lipgloss.Color
	surfaceDeep  lipgloss.Color
	border       lipgloss.Color
	borderStrong lipgloss.Color
	text         lipgloss.Color
	textSoft     lipgloss.Color
	textMuted    lipgloss.Color
	accent       lipgloss.Color
	accentDeep   lipgloss.Color
	success      lipgloss.Color
	warn         lipgloss.Color
	danger       lipgloss.Color
}

func newChronicleTheme() chronicleTheme {
	return chronicleTheme{
		surface:      lipgloss.Color("236"),
		surfaceAlt:   lipgloss.Color("235"),
		surfaceDeep:  lipgloss.Color("238"),
		border:       lipgloss.Color("239"),
		borderStrong: lipgloss.Color("25"),
		text:         lipgloss.Color("252"),
		textSoft:     lipgloss.Color("250"),
		textMuted:    lipgloss.Color("244"),
		accent:       lipgloss.Color("39"),
		accentDeep:   lipgloss.Color("25"),
		success:      lipgloss.Color("42"),
		warn:         lipgloss.Color("179"),
		danger:       lipgloss.Color("167"),
	}
}

func (t chronicleTheme) panel(width int, height int, accent bool) lipgloss.Style {
	border := t.border
	if accent {
		border = t.borderStrong
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Background(t.surfaceAlt).
		Padding(1)
	if width > 0 {
		style = style.Width(max(0, width-2))
	}
	if height > 0 {
		style = style.Height(max(0, height-2))
	}
	return style
}

func (t chronicleTheme) modal(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(max(0, width-2)).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(t.accent).
		Background(t.surfaceAlt).
		Padding(1)
}

func (t chronicleTheme) masthead(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(max(0, width-2)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.borderStrong).
		Background(t.surface).
		Padding(1)
}

func (t chronicleTheme) promptBar(width int, focused bool) lipgloss.Style {
	border := t.border
	if focused {
		border = t.accent
	}
	return lipgloss.NewStyle().
		Width(max(0, width-2)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Background(t.surface).
		Padding(0, 1)
}

func (t chronicleTheme) footer(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Foreground(t.textMuted)
}

func (t chronicleTheme) title() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(t.text)
}

func (t chronicleTheme) sectionTitle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(t.textSoft)
}

func (t chronicleTheme) body() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.text)
}

func (t chronicleTheme) muted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.textMuted)
}

func (t chronicleTheme) subtle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.textSoft)
}

func (t chronicleTheme) badge(label string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(t.accentDeep).
		Padding(0, 1).
		Render(label)
}

func (t chronicleTheme) chip(label string, active bool, accent bool) string {
	style := lipgloss.NewStyle().
		Foreground(t.textMuted).
		Background(t.surfaceDeep).
		Padding(0, 1)
	if active {
		if accent {
			style = style.Foreground(lipgloss.Color("230")).Background(t.accentDeep)
		} else {
			style = style.Foreground(t.text).Background(t.surface)
		}
	}
	return style.Render(label)
}

func (t chronicleTheme) keycap(label string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(t.accentDeep).
		Padding(0, 1).
		Bold(true).
		Render(label)
}

func (t chronicleTheme) inputBox(focused bool) lipgloss.Style {
	border := t.border
	if focused {
		border = t.accent
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Background(t.surface).
		Padding(0, 1)
}

func (t chronicleTheme) selectedRow() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.text).Background(t.surfaceDeep)
}

func (t chronicleTheme) statusText(tone chronicleStatusTone) lipgloss.Style {
	color := t.textMuted
	switch tone {
	case chronicleStatusSuccess:
		color = t.success
	case chronicleStatusWarn:
		color = t.warn
	case chronicleStatusError:
		color = t.danger
	default:
		color = t.accent
	}
	return lipgloss.NewStyle().Foreground(color)
}
