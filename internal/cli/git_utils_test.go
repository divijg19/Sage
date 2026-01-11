package cli

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func hasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func TestGitHooksDir_Default(t *testing.T) {
	if !hasGit() {
		t.Skip("git not available")
	}

	repo := t.TempDir()
	runGit(t, repo, "init")

	dir, core, err := gitHooksDir(repo)
	if err != nil {
		t.Fatalf("gitHooksDir: %v", err)
	}
	if core != "" {
		t.Fatalf("expected no core.hooksPath, got %q", core)
	}
	want := filepath.Join(repo, ".git", "hooks")
	if dir != want {
		t.Fatalf("expected %q, got %q", want, dir)
	}
}

func TestGitHooksDir_CoreHooksPathRelative(t *testing.T) {
	if !hasGit() {
		t.Skip("git not available")
	}

	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "core.hooksPath", ".githooks")

	dir, core, err := gitHooksDir(repo)
	if err != nil {
		t.Fatalf("gitHooksDir: %v", err)
	}
	if core != ".githooks" {
		t.Fatalf("expected core.hooksPath '.githooks', got %q", core)
	}
	want := filepath.Join(repo, ".githooks")
	if dir != want {
		t.Fatalf("expected %q, got %q", want, dir)
	}
}
