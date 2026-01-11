package cli

import (
	"path/filepath"
	"testing"
)

func TestNormalizeProjectName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"   ", ""},
		{"My App", "my-app"},
		{"  My App  ", "my-app"},
		{"---My App---", "my-app"},
		{"__My App__", "my-app"},
		{"my-app", "my-app"},
	}

	for _, c := range cases {
		if got := normalizeProjectName(c.in); got != c.want {
			t.Fatalf("normalizeProjectName(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestResolveProjectFilter_Precedence(t *testing.T) {
	t.Setenv("SAGE_PROJECT", "envproj")

	if p, ok := resolveProjectFilter("", false); !ok || p != "envproj" {
		t.Fatalf("expected env filter envproj=true, got %q,%v", p, ok)
	}

	if p, ok := resolveProjectFilter("explicit", false); !ok || p != "explicit" {
		t.Fatalf("expected explicit filter explicit=true, got %q,%v", p, ok)
	}

	if p, ok := resolveProjectFilter("explicit", true); ok || p != "" {
		t.Fatalf("expected all=true disables filter, got %q,%v", p, ok)
	}
}

func TestSuggestedProjectFromRepo_FallsBackToBasename(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "My Repo")

	got := suggestedProjectFromRepo(repo)
	if got != "my-repo" {
		t.Fatalf("expected my-repo, got %q", got)
	}
}
