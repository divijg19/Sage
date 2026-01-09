package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAll_SortsByFilename(t *testing.T) {
	dir := t.TempDir()

	// Intentionally write out of order
	if err := os.WriteFile(filepath.Join(dir, "02-decision.md"), []byte("# two"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "01-record.md"), []byte("# one"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(got))
	}
	if got[0].Name != "01-record" {
		t.Fatalf("expected first template 01-record, got %q", got[0].Name)
	}
	if got[1].Name != "02-decision" {
		t.Fatalf("expected second template 02-decision, got %q", got[1].Name)
	}
}

func TestParseTemplate_SuggestedKindFrontMatter(t *testing.T) {
	raw := "---\nsuggested_kind: decision\n---\n\n# Body"
	tpl := parseTemplate("decision.md", raw)
	if tpl.Name != "decision" {
		t.Fatalf("expected name decision, got %q", tpl.Name)
	}
	if tpl.SuggestedKind != "decision" {
		t.Fatalf("expected suggested_kind decision, got %q", tpl.SuggestedKind)
	}
	if tpl.Body != "# Body" {
		t.Fatalf("expected trimmed body, got %q", tpl.Body)
	}
}
