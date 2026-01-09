package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"
)

// openEditor opens the user's editor with a template
// and returns the resulting content verbatim.
// If the editor is aborted (Ctrl+C), it returns an empty string and nil error.
func openEditor(template string) (string, error) {
	editor, err := resolveEditorCommand()
	if err != nil {
		return "", err
	}
	if editor == "" {
		editor = "vi"
	}

	argv := strings.Fields(editor)
	if len(argv) == 0 {
		argv = []string{"vi"}
	}
	bin := argv[0]
	extraArgs := argv[1:]

	extraArgs = ensureEditorWaitArgs(bin, extraArgs)

	tmpFile, err := os.CreateTemp("", "sage-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(template); err != nil {
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	cmd := exec.Command(bin, append(extraArgs, tmpFile.Name())...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Treat editor abort as graceful cancel
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", nil
		}
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("editor not found: %s", bin)
		}
		return "", err
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func resolveEditorCommand() (string, error) {
	configured, err := getConfiguredEditor()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(configured) != "" {
		return configured, nil
	}
	if v := strings.TrimSpace(os.Getenv("SAGE_EDITOR")); v != "" {
		return v, nil
	}
	return strings.TrimSpace(os.Getenv("EDITOR")), nil
}

func hasWaitFlag(args []string) bool {
	for _, a := range args {
		switch a {
		case "--wait", "-w":
			return true
		}
	}
	return false
}

func hasAnyFlag(args []string, flags ...string) bool {
	if len(args) == 0 || len(flags) == 0 {
		return false
	}
	for _, a := range args {
		for _, f := range flags {
			if a == f {
				return true
			}
		}
	}
	return false
}

func ensureEditorWaitArgs(bin string, extraArgs []string) []string {
	// Some GUI editors return immediately unless told to wait.
	switch bin {
	case "code", "code-insiders", "codium", "zed", "subl", "gedit", "gnome-text-editor":
		if !hasWaitFlag(extraArgs) {
			return append(extraArgs, "--wait")
		}
		return extraArgs
	case "kate":
		if !hasAnyFlag(extraArgs, "--block") {
			return append(extraArgs, "--block")
		}
		return extraArgs
	default:
		return extraArgs
	}
}

func prepareEditorBody(tpl string, title string) string {
	if tpl == "" {
		return ""
	}
	return strings.ReplaceAll(tpl, "{{title}}", title)
}

func resolveEditorKindSeed(explicitKind string, suggested string) string {
	if explicitKind == "decision" || explicitKind == "d" {
		return "decision"
	}
	if explicitKind == "record" || explicitKind == "r" {
		return "record"
	}
	if suggested == "decision" {
		return "decision"
	}
	return "record"
}

func ensureFrontMatter(body string, title string, kind string) string {
	trimmed := strings.TrimLeft(body, "\n\r\t ")
	if strings.HasPrefix(trimmed, "---\n") || strings.HasPrefix(trimmed, "---\r\n") || trimmed == "---" {
		// Try to inject missing fields without disturbing existing metadata too much.
		lines := strings.Split(trimmed, "\n")
		end := -1
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				end = i
				break
			}
		}
		if end == -1 {
			return trimmed
		}

		hasTitle := false
		hasKind := false
		for i := 1; i < end; i++ {
			l := strings.TrimSpace(lines[i])
			if strings.HasPrefix(l, "title:") {
				lines[i] = fmt.Sprintf("title: %s", yamlQuote(title))
				hasTitle = true
				continue
			}
			if strings.HasPrefix(l, "kind:") {
				if kind != "" {
					lines[i] = fmt.Sprintf("kind: %s", kind)
				}
				hasKind = true
				continue
			}
		}

		insert := []string{}
		if !hasTitle {
			insert = append(insert, fmt.Sprintf("title: %s", yamlQuote(title)))
		}
		if !hasKind && kind != "" {
			insert = append(insert, fmt.Sprintf("kind: %s", kind))
		}
		if len(insert) == 0 {
			return strings.Join(lines, "\n")
		}

		newLines := []string{}
		newLines = append(newLines, lines[:1]...)
		newLines = append(newLines, insert...)
		newLines = append(newLines, lines[1:]...)
		return strings.Join(newLines, "\n")
	}

	return fmt.Sprintf(
		"---\ntitle: %s\nkind: %s\n---\n\n%s\n",
		yamlQuote(title),
		kind,
		strings.TrimSpace(body),
	)
}

func defaultEditorTemplate(explicitKind string) string {
	if explicitKind == "decision" || explicitKind == "d" {
		return `---
title: "{{title}}"
kind: decision
---

# Decision

## Context

## Options

## Decision

## Consequences
`
	}

	return `---
title: "{{title}}"
kind: record
---

# Notes

## Context

## What I did

## Next steps
`
}

// extractTitleAndBodyFromEditor looks for YAML front matter at the top of the
// buffer and extracts a `title:` field (if present). It returns the extracted
// title (possibly empty) and the remaining body.
func extractMetaAndBodyFromEditor(raw string) (string, string, string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", ""
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", stripBoilerplate(raw)
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return "", "", stripBoilerplate(raw)
	}

	title := ""
	kind := ""
	for _, line := range lines[1:end] {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "title:") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			title = strings.Trim(title, `"'`)
			continue
		}
		if strings.HasPrefix(line, "kind:") {
			kind = strings.TrimSpace(strings.TrimPrefix(line, "kind:"))
			kind = strings.ToLower(strings.Trim(kind, `"'`))
		}
	}

	body := strings.Join(lines[end+1:], "\n")
	body = stripBoilerplate(body)
	return title, kind, body
}

func yamlQuote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return "\"" + s + "\""
}

func stripBoilerplate(s string) string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "<!--") && strings.Contains(trim, "sage:") {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func normalizeForComparison(raw string) string {
	_, _, body := extractMetaAndBodyFromEditor(raw)
	return normalizePlainText(body)
}

func normalizePlainText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// isMeaningfulContent is intentionally conservative: it requires at least one
// non-heading line containing a letter or number.
func isMeaningfulContent(body string) bool {
	body = strings.TrimSpace(body)
	if body == "" {
		return false
	}

	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if strings.HasPrefix(trim, "#") {
			continue
		}
		if strings.HasPrefix(trim, "<!--") {
			continue
		}

		for _, r := range trim {
			if unicode.IsLetter(r) || unicode.IsNumber(r) {
				return true
			}
		}
	}

	return false
}
