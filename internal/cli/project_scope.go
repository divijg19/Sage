package cli

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultProjectName = "global"

func normalizeProjectName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.Trim(s, "-_")
	return s
}

func activeProjectFromEnv() string {
	return normalizeProjectName(os.Getenv("SAGE_PROJECT"))
}

func projectForNewEntry() string {
	if p := activeProjectFromEnv(); p != "" {
		return p
	}
	return defaultProjectName
}

// resolveProjectFilter returns (project, filterEnabled).
// Precedence: explicitProject > (active project env) > no filter.
func resolveProjectFilter(explicitProject string, all bool) (string, bool) {
	if all {
		return "", false
	}
	if p := normalizeProjectName(explicitProject); p != "" {
		return p, true
	}
	if p := activeProjectFromEnv(); p != "" {
		return p, true
	}
	return "", false
}

func suggestedProjectFromRepo(repo string) string {
	if strings.TrimSpace(repo) == "" {
		if cwd, err := os.Getwd(); err == nil {
			repo = cwd
		}
	}

	root, err := gitRepoRoot(repo)
	if err == nil && strings.TrimSpace(root) != "" {
		return normalizeProjectName(filepath.Base(root))
	}

	if strings.TrimSpace(repo) != "" {
		return normalizeProjectName(filepath.Base(repo))
	}
	return ""
}
