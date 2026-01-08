package project

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Detect() (string, string, error) {
	root, err := gitRoot()
	if err == nil {
		name := repoName(root)
		path, err := storagePath(name)
		return name, path, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}

	name := "cwd-" + hashPath(cwd)
	path, err := storagePath(name)
	return name, path, err
}

func gitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func repoName(root string) string {
	return filepath.Base(root)
}

func hashPath(path string) string {
	h := sha1.Sum([]byte(path))
	return hex.EncodeToString(h[:8])
}

func storagePath(project string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".sage", project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	return filepath.Join(dir, "sage.db"), nil
}
