package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/divijg19/sage/internal/event"
	"github.com/divijg19/sage/internal/store"
)

func globalDBPath() string {
	dir := sageDir()
	if dir == "" {
		return ""
	}
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "sage.db")
}

func openGlobalStore() (*store.Store, error) {
	path := globalDBPath()
	if path == "" {
		return nil, fmt.Errorf("could not determine Sage directory")
	}

	s, err := store.Open(path)
	if err != nil {
		return nil, err
	}

	// Import legacy per-directory stores only if the global DB is empty.
	// This makes the migration safe and idempotent.
	if err := maybeImportLegacyStores(s); err != nil {
		return nil, err
	}

	return s, nil
}

func maybeImportLegacyStores(s *store.Store) error {
	count, err := s.Count()
	if err != nil {
		return err
	}
	if count != 0 {
		return nil
	}

	paths, err := legacyDBPaths()
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return nil
	}

	var all []event.Event
	for _, p := range paths {
		evts, err := store.ReadEventsFromDB(p)
		if err != nil {
			// Skip unreadable legacy stores rather than failing the whole CLI.
			continue
		}
		all = append(all, evts...)
	}

	inserted, err := s.ImportEvents(all)
	if err != nil {
		return err
	}
	if inserted > 0 {
		fmt.Fprintf(os.Stderr, "Imported %d legacy entries into global store.\n", inserted)
	}
	return nil
}

func legacyDBPaths() ([]string, error) {
	root := sageDir()
	if root == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, de := range entries {
		if !de.IsDir() {
			continue
		}
		name := de.Name()
		if name == "templates" {
			continue
		}
		p := filepath.Join(root, name, "sage.db")
		if _, err := os.Stat(p); err == nil {
			out = append(out, p)
		}
	}

	return out, nil
}
