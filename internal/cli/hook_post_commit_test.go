package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/divijg19/sage/internal/event"
)

func TestRunHookPostCommit_AppendsOnceDeterministic(t *testing.T) {
	if !hasGit() {
		t.Skip("git not available")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "Sage Test")
	runGit(t, repo, "config", "user.email", "sage@example.com")

	// Create a commit.
	if err := os.WriteFile(repo+"/file.txt", []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "test commit")

	// Run twice; deterministic ID should dedupe.
	_ = runHookPostCommit(repo)
	_ = runHookPostCommit(repo)

	s, err := openGlobalStore()
	if err != nil {
		t.Fatalf("openGlobalStore: %v", err)
	}

	events, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != event.CommitKind {
		t.Fatalf("expected kind %q, got %q", event.CommitKind, events[0].Kind)
	}
	if events[0].Title == "" {
		t.Fatalf("expected title")
	}
	if events[0].Metadata["sha"] == "" {
		t.Fatalf("expected sha metadata")
	}
	if events[0].Metadata["repo_root"] == "" {
		t.Fatalf("expected repo_root metadata")
	}
}

func TestHookScriptInvokesSageWithRepo(t *testing.T) {
	// Quick sanity: the hook script should pass --repo to avoid ambiguity.
	dir := t.TempDir()
	res, err := InstallHook(dir, "post-commit", HookInstallOptions{})
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	_ = res

	b, err := os.ReadFile(filepath.Join(dir, "post-commit"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !containsAll(string(b), []string{"sage hook post-commit", "--repo"}) {
		t.Fatalf("expected hook script to pass --repo")
	}
}

func containsAll(s string, parts []string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
