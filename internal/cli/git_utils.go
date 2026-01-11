package cli

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func gitOutput(repo string, args ...string) (string, error) {
	var cmd *exec.Cmd
	if strings.TrimSpace(repo) == "" {
		cmd = exec.Command("git", args...)
	} else {
		cmd = exec.Command("git", append([]string{"-C", repo}, args...)...)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func gitRepoRoot(repo string) (string, error) {
	out, err := gitOutput(repo, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return out, nil
}

func gitDirAbs(repo string) (string, error) {
	root, err := gitRepoRoot(repo)
	if err != nil {
		return "", err
	}

	gd, err := gitOutput(repo, "rev-parse", "--git-dir")
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(gd) {
		return gd, nil
	}
	return filepath.Join(root, gd), nil
}

func gitHooksDir(repo string) (string, string, error) {
	// If core.hooksPath is set, it may be absolute or relative to the repo root.
	hooksPath, _ := gitOutput(repo, "config", "--get", "core.hooksPath")
	if strings.TrimSpace(hooksPath) != "" {
		root, err := gitRepoRoot(repo)
		if err != nil {
			return "", "", err
		}
		if filepath.IsAbs(hooksPath) {
			return hooksPath, hooksPath, nil
		}
		return filepath.Join(root, hooksPath), hooksPath, nil
	}

	gd, err := gitDirAbs(repo)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(gd, "hooks"), "", nil
}
