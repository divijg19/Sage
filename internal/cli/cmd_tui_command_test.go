package cli

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type stubChronicleProgram struct {
	model tea.Model
	err   error
}

func (p stubChronicleProgram) Run() (tea.Model, error) {
	return p.model, p.err
}

func TestChronicleOptionsFromFlags_NormalizesQueryAndTags(t *testing.T) {
	tuiAll = false
	tuiProject = "alpha"
	tuiTags = []string{"Auth, backend", "backend"}
	tuiQuery = "  cache drift  "
	t.Setenv("SAGE_PROJECT", "")

	opts := chronicleOptionsFromFlags()
	if opts.Project != "alpha" {
		t.Fatalf("expected project alpha, got %q", opts.Project)
	}
	if opts.Query != "cache drift" {
		t.Fatalf("expected trimmed query, got %q", opts.Query)
	}
	if len(opts.Tags) != 2 || opts.Tags[0] != "auth" || opts.Tags[1] != "backend" {
		t.Fatalf("unexpected normalized tags: %#v", opts.Tags)
	}
}

func TestChronicleOptionsFromFlags_AllOverridesProjectAndEnv(t *testing.T) {
	tuiAll = true
	tuiProject = "explicit"
	tuiTags = nil
	tuiQuery = ""
	t.Setenv("SAGE_PROJECT", "envproj")

	opts := chronicleOptionsFromFlags()
	if opts.Project != "" {
		t.Fatalf("expected all scope to clear project, got %q", opts.Project)
	}
}

func TestTUIRunE_SeedsModelFromFlags(t *testing.T) {
	oldFactory := newChronicleProgram
	defer func() { newChronicleProgram = oldFactory }()

	var captured chronicleModel
	newChronicleProgram = func(model tea.Model) chronicleProgram {
		captured = model.(chronicleModel)
		return stubChronicleProgram{model: model}
	}

	tuiAll = false
	tuiProject = "beta"
	tuiTags = []string{"Ops, Infra"}
	tuiQuery = "  rollout plan  "
	t.Setenv("SAGE_PROJECT", "")

	if err := tuiCmd.RunE(tuiCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	if captured.selectedProject != "beta" {
		t.Fatalf("expected selected project beta, got %q", captured.selectedProject)
	}
	if captured.query != "rollout plan" {
		t.Fatalf("expected trimmed query, got %q", captured.query)
	}
	if !captured.tagFilter["ops"] || !captured.tagFilter["infra"] {
		t.Fatalf("expected normalized tags in initial filter: %#v", captured.tagFilter)
	}
}
