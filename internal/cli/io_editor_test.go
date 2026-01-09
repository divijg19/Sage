package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveEditorCommand_Precedence_ConfigOverEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EDITOR", "vim")
	t.Setenv("SAGE_EDITOR", "nano")

	if err := setConfiguredEditor("micro"); err != nil {
		t.Fatalf("setConfiguredEditor: %v", err)
	}

	got, err := resolveEditorCommand()
	if err != nil {
		t.Fatalf("resolveEditorCommand: %v", err)
	}
	if got != "micro" {
		t.Fatalf("expected config editor 'micro', got %q", got)
	}
}

func TestResolveEditorCommand_Precedence_SAGE_EDITOROverEDITOR(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EDITOR", "vim")
	t.Setenv("SAGE_EDITOR", "nano")

	got, err := resolveEditorCommand()
	if err != nil {
		t.Fatalf("resolveEditorCommand: %v", err)
	}
	if got != "nano" {
		t.Fatalf("expected SAGE_EDITOR 'nano', got %q", got)
	}
}

func TestResolveEditorCommand_Precedence_EDITORFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("EDITOR", "vim")
	t.Setenv("SAGE_EDITOR", "")

	got, err := resolveEditorCommand()
	if err != nil {
		t.Fatalf("resolveEditorCommand: %v", err)
	}
	if got != "vim" {
		t.Fatalf("expected EDITOR 'vim', got %q", got)
	}
}

func TestEnsureFrontMatter_InjectsAndQuotesTitle(t *testing.T) {
	body := "# Notes\n\nSomething"
	out := ensureFrontMatter(body, `Title: with "quotes"`, "record")

	if !strings.HasPrefix(out, "---\n") {
		t.Fatalf("expected front matter start, got: %q", out)
	}
	if !strings.Contains(out, "title: \"Title: with \\\"quotes\\\"\"\n") {
		t.Fatalf("expected quoted title, got:\n%s", out)
	}
	if !strings.Contains(out, "kind: record\n") {
		t.Fatalf("expected kind, got:\n%s", out)
	}
}

func TestEnsureFrontMatter_RewritesExistingTitle(t *testing.T) {
	in := "---\ntitle: old\nkind: record\n---\n\n# Notes\n"
	out := ensureFrontMatter(in, "new title", "record")
	if !strings.Contains(out, "title: \"new title\"\n") {
		t.Fatalf("expected title rewrite, got:\n%s", out)
	}
}

func TestConfig_SaveLoad_Unset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := setConfiguredEditor("nano"); err != nil {
		t.Fatalf("setConfiguredEditor: %v", err)
	}

	path := configPath()
	if path == "" {
		t.Fatalf("expected configPath")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	got, err := getConfiguredEditor()
	if err != nil {
		t.Fatalf("getConfiguredEditor: %v", err)
	}
	if got != "nano" {
		t.Fatalf("expected 'nano', got %q", got)
	}

	if err := unsetConfiguredEditor(); err != nil {
		t.Fatalf("unsetConfiguredEditor: %v", err)
	}

	got, err = getConfiguredEditor()
	if err != nil {
		t.Fatalf("getConfiguredEditor: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty after unset, got %q", got)
	}

	// Ensure file remains inside ~/.sage
	if filepath.Dir(path) != filepath.Join(home, ".sage") {
		t.Fatalf("unexpected config dir: %s", filepath.Dir(path))
	}
}

func TestHasWaitFlag(t *testing.T) {
	if hasWaitFlag([]string{"--wait"}) != true {
		t.Fatalf("expected true")
	}
	if hasWaitFlag([]string{"-w"}) != true {
		t.Fatalf("expected true")
	}
	if hasWaitFlag([]string{"--foo"}) != false {
		t.Fatalf("expected false")
	}
}

func TestEnsureEditorWaitArgs(t *testing.T) {
	{
		out := ensureEditorWaitArgs("zed", nil)
		if !hasWaitFlag(out) {
			t.Fatalf("expected zed to get --wait, got %v", out)
		}
	}
	{
		out := ensureEditorWaitArgs("code", []string{"--wait"})
		count := 0
		for _, a := range out {
			if a == "--wait" {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("expected a single --wait, got %v", out)
		}
	}
	{
		out := ensureEditorWaitArgs("kate", nil)
		if !hasAnyFlag(out, "--block") {
			t.Fatalf("expected kate to get --block, got %v", out)
		}
	}
	{
		out := ensureEditorWaitArgs("vim", nil)
		if len(out) != 0 {
			t.Fatalf("expected vim args unchanged, got %v", out)
		}
	}
}
