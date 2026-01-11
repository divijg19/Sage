package cli

import "testing"

func TestConfig_Tags_EnsureConfigured(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := ensureTagsConfigured([]string{"auth", "backend"}); err != nil {
		t.Fatalf("ensureTagsConfigured: %v", err)
	}
	if err := ensureTagsConfigured([]string{"auth"}); err != nil {
		t.Fatalf("ensureTagsConfigured (dedupe): %v", err)
	}

	tags, err := getConfiguredTags()
	if err != nil {
		t.Fatalf("getConfiguredTags: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", tags)
	}
	if tags[0] != "auth" || tags[1] != "backend" {
		t.Fatalf("expected [auth backend], got %v", tags)
	}
}
