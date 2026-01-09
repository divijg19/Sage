package cli

import "testing"

func TestParseTags_NormalizesAndDedupes(t *testing.T) {
	got := parseTags([]string{"Auth,backend", " backend ", "", "Infra"})
	// parseTags preserves first-seen order after normalization
	want := []string{"auth", "backend", "infra"}
	if len(got) != len(want) {
		t.Fatalf("expected %d tags, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %q at %d, got %q", want[i], i, got[i])
		}
	}
}
