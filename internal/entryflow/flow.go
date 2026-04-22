package entryflow

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/divijg19/sage/internal/event"
)

type Status string

const (
	StatusCanceled  Status = "canceled"
	StatusUnchanged Status = "unchanged"
	StatusEmpty     Status = "empty"
	StatusDuplicate Status = "duplicate"
	StatusSaved     Status = "saved"
)

type Store interface {
	Append(e event.Event) error
	Latest() (*event.Event, error)
	LatestByProject(project string) (*event.Event, error)
}

type InitialBuffer struct {
	Body     string
	SeedKind string
}

type FinalizeRequest struct {
	Title         string
	ExplicitKind  string
	SuggestedKind string
	SeedKind      string
	InitialBody   string
	Edited        string
	Project       string
	Tags          []string
}

type Dependencies struct {
	Store       Store
	EnsureTags  func([]string) error
	ResolveKind func(explicit string, suggested string) (event.EntryKind, error)
	ConfirmSave func() bool
	Now         func() time.Time
	NewID       func() string
}

type Result struct {
	Status Status
	Event  *event.Event
}

func PrepareInitialBuffer(title string, explicitKind string, suggestedKind string, templateBody string) InitialBuffer {
	seedKind := ResolveEditorKindSeed(explicitKind, suggestedKind)
	body := templateBody
	if strings.TrimSpace(body) == "" {
		body = DefaultEditorTemplate(explicitKind)
	}

	body = EnsureFrontMatter(
		PrepareEditorBody(body, title),
		title,
		seedKind,
	)

	return InitialBuffer{
		Body:     body,
		SeedKind: seedKind,
	}
}

func Finalize(req FinalizeRequest, deps Dependencies) (Result, error) {
	if strings.TrimSpace(req.Edited) == "" {
		return Result{Status: StatusCanceled}, nil
	}

	if NormalizeForComparison(req.Edited) == NormalizeForComparison(req.InitialBody) {
		return Result{Status: StatusUnchanged}, nil
	}

	title, editedKind, cleaned := ExtractMetaAndBodyFromEditor(req.Edited)
	if strings.TrimSpace(title) == "" {
		title = req.Title
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return Result{}, fmt.Errorf("title is required")
	}

	explicitKind := req.ExplicitKind
	if editedKind != "" && !strings.EqualFold(editedKind, req.SeedKind) {
		explicitKind = editedKind
	}

	content := strings.TrimSpace(cleaned)
	if !IsMeaningfulContent(content) {
		return Result{Status: StatusEmpty}, nil
	}

	if deps.EnsureTags != nil {
		if err := deps.EnsureTags(req.Tags); err != nil {
			return Result{}, err
		}
	}

	if deps.ResolveKind == nil {
		return Result{}, fmt.Errorf("resolve kind callback is required")
	}

	kind, err := deps.ResolveKind(explicitKind, req.SuggestedKind)
	if err != nil {
		return Result{}, err
	}

	if deps.ConfirmSave != nil && !deps.ConfirmSave() {
		return Result{Status: StatusCanceled}, nil
	}

	if deps.Store == nil {
		return Result{}, fmt.Errorf("store is required")
	}

	prev, err := latestForProject(deps.Store, req.Project)
	if err != nil {
		return Result{}, err
	}
	if prev != nil &&
		prev.Kind == kind &&
		strings.TrimSpace(prev.Title) == title &&
		NormalizePlainText(prev.Content) == NormalizePlainText(content) &&
		normalizeTagSet(prev.Tags) == normalizeTagSet(req.Tags) {
		return Result{Status: StatusDuplicate}, nil
	}

	now := time.Now
	if deps.Now != nil {
		now = deps.Now
	}

	newID := uuid.NewString
	if deps.NewID != nil {
		newID = deps.NewID
	}

	e := event.Event{
		ID:        newID(),
		Timestamp: now(),
		Project:   req.Project,
		Kind:      kind,
		Title:     title,
		Content:   content,
		Tags:      append([]string(nil), req.Tags...),
	}

	if err := deps.Store.Append(e); err != nil {
		return Result{}, err
	}

	if saved, err := latestForProject(deps.Store, req.Project); err == nil && saved != nil && saved.ID == e.ID {
		e.Seq = saved.Seq
	}

	return Result{
		Status: StatusSaved,
		Event:  &e,
	}, nil
}

func latestForProject(s Store, project string) (*event.Event, error) {
	if strings.TrimSpace(project) != "" {
		return s.LatestByProject(project)
	}
	return s.Latest()
}

func PrepareEditorBody(tpl string, title string) string {
	if tpl == "" {
		return ""
	}
	return strings.ReplaceAll(tpl, "{{title}}", title)
}

func ResolveEditorKindSeed(explicitKind string, suggested string) string {
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

func EnsureFrontMatter(body string, title string, kind string) string {
	trimmed := strings.TrimLeft(body, "\n\r\t ")
	if strings.HasPrefix(trimmed, "---\n") || strings.HasPrefix(trimmed, "---\r\n") || trimmed == "---" {
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

		var insert []string
		if !hasTitle {
			insert = append(insert, fmt.Sprintf("title: %s", yamlQuote(title)))
		}
		if !hasKind && kind != "" {
			insert = append(insert, fmt.Sprintf("kind: %s", kind))
		}
		if len(insert) == 0 {
			return strings.Join(lines, "\n")
		}

		newLines := append([]string{}, lines[:1]...)
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

func DefaultEditorTemplate(explicitKind string) string {
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

func ExtractMetaAndBodyFromEditor(raw string) (string, string, string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", ""
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", StripBoilerplate(raw)
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return "", "", StripBoilerplate(raw)
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
	body = StripBoilerplate(body)
	return title, kind, body
}

func StripBoilerplate(s string) string {
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

func NormalizeForComparison(raw string) string {
	_, _, body := ExtractMetaAndBodyFromEditor(raw)
	return NormalizePlainText(body)
}

func NormalizePlainText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func IsMeaningfulContent(body string) bool {
	body = strings.TrimSpace(body)
	if body == "" {
		return false
	}

	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") || strings.HasPrefix(trim, "<!--") {
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

func yamlQuote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return "\"" + s + "\""
}

func normalizeTagSet(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	copyTags := append([]string(nil), tags...)
	sort.Strings(copyTags)
	return strings.Join(copyTags, ",")
}
