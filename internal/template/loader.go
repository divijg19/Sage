package template

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func LoadAll(dir string) ([]Template, error) {
	var templates []Template

	entries, err := os.ReadDir(dir)
	if err != nil {
		return templates, nil // empty is OK
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		path := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		t := parseTemplate(e.Name(), string(b))
		templates = append(templates, t)
	}

	return templates, nil
}
