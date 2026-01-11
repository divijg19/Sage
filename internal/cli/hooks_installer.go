package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type HookInstallOptions struct {
	Force  bool
	DryRun bool
	Sync   bool
}

type HookInspect struct {
	HookPath       string
	Exists         bool
	SageManaged    bool
	LegacyHookPath string
}

func InspectHook(hooksDir, hookName string) (HookInspect, error) {
	hookPath := filepath.Join(hooksDir, hookName)
	b, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return HookInspect{HookPath: hookPath, Exists: false}, nil
		}
		return HookInspect{}, err
	}

	content := string(b)
	legacy := parseLegacyHookPath(content)
	return HookInspect{
		HookPath:       hookPath,
		Exists:         true,
		SageManaged:    isSageManagedHook(content, hookName),
		LegacyHookPath: legacy,
	}, nil
}

type HookInstallResult struct {
	HookPath      string
	BackedUp      bool
	BackupPath    string
	Installed     bool
	Updated       bool
	WasManaged    bool
	LegacyChained bool
}

func InstallHook(hooksDir, hookName string, opts HookInstallOptions) (HookInstallResult, error) {
	if strings.TrimSpace(hooksDir) == "" {
		return HookInstallResult{}, fmt.Errorf("hooks directory is required")
	}
	if strings.TrimSpace(hookName) == "" {
		return HookInstallResult{}, fmt.Errorf("hook name is required")
	}

	hookPath := filepath.Join(hooksDir, hookName)
	inspected, err := InspectHook(hooksDir, hookName)
	if err != nil {
		return HookInstallResult{}, err
	}

	res := HookInstallResult{HookPath: hookPath, WasManaged: inspected.SageManaged}

	var legacyPath string
	if inspected.Exists && !inspected.SageManaged {
		if opts.Force {
			if !opts.DryRun {
				_ = os.Remove(hookPath)
			}
		} else {
			backupPath, err := backupHookFile(hookPath, hookName, opts.DryRun)
			if err != nil {
				return HookInstallResult{}, err
			}
			legacyPath = backupPath
			res.BackedUp = true
			res.BackupPath = backupPath
			res.LegacyChained = true
		}
	} else if inspected.SageManaged {
		legacyPath = inspected.LegacyHookPath
		res.LegacyChained = strings.TrimSpace(legacyPath) != ""
	}

	script := renderHookScript(hookName, legacyPath, opts.Sync)

	if !opts.DryRun {
		if err := os.MkdirAll(hooksDir, 0o755); err != nil {
			return HookInstallResult{}, err
		}

		prev, _ := os.ReadFile(hookPath)
		had := len(prev) > 0
		if err := os.WriteFile(hookPath, []byte(script), 0o755); err != nil {
			return HookInstallResult{}, err
		}
		_ = os.Chmod(hookPath, 0o755)

		if had {
			res.Updated = true
		} else {
			res.Installed = true
		}
	}

	return res, nil
}

func UninstallHook(hooksDir, hookName string, opts HookInstallOptions) (string, error) {
	inspected, err := InspectHook(hooksDir, hookName)
	if err != nil {
		return "", err
	}
	if !inspected.Exists {
		return "not installed", nil
	}
	if !inspected.SageManaged {
		return "existing hook is not Sage-managed; refusing to modify", nil
	}

	legacy := inspected.LegacyHookPath
	if strings.TrimSpace(legacy) != "" {
		if _, err := os.Stat(legacy); err == nil {
			if !opts.DryRun {
				if err := os.Rename(legacy, inspected.HookPath); err != nil {
					return "", err
				}
			}
			return "restored legacy hook", nil
		}
	}

	if !opts.DryRun {
		if err := os.Remove(inspected.HookPath); err != nil {
			return "", err
		}
	}
	return "removed Sage hook", nil
}

func backupHookFile(hookPath, hookName string, dryRun bool) (string, error) {
	dir := filepath.Dir(hookPath)
	base := hookName + ".sage.legacy"
	candidate := filepath.Join(dir, base)
	if _, err := os.Stat(candidate); err == nil {
		candidate = filepath.Join(dir, base+"."+strconv.FormatInt(time.Now().Unix(), 10))
	}

	if !dryRun {
		if err := os.Rename(hookPath, candidate); err != nil {
			return "", err
		}
	}
	return candidate, nil
}

func isSageManagedHook(content string, hookName string) bool {
	needle := "# sage-hook: " + hookName + " v1"
	return strings.Contains(content, needle)
}

var legacyHookRe = regexp.MustCompile(`(?m)^LEGACY_HOOK=(?:"([^"]*)"|'([^']*)'|([^\s#]*))\s*$`)

func parseLegacyHookPath(content string) string {
	m := legacyHookRe.FindStringSubmatch(content)
	if len(m) == 0 {
		return ""
	}
	for i := 1; i < len(m); i++ {
		if strings.TrimSpace(m[i]) != "" {
			return strings.TrimSpace(m[i])
		}
	}
	return ""
}

func renderHookScript(hookName, legacyHookPath string, sync bool) string {
	legacyLine := "LEGACY_HOOK=\"\""
	if strings.TrimSpace(legacyHookPath) != "" {
		legacyLine = "LEGACY_HOOK=\"" + escapeForDoubleQuotes(legacyHookPath) + "\""
	}

	var sageInvoke string
	if sync {
		sageInvoke = "sage hook " + hookName + " --repo \"$REPO_DIR\" >/dev/null 2>&1 || true"
	} else {
		sageInvoke = "( sage hook " + hookName + " --repo \"$REPO_DIR\" >/dev/null 2>&1 || true ) &"
	}

	return strings.TrimSpace(fmt.Sprintf(`#!/bin/sh
# sage-hook: %s v1

# Never block commits. If anything fails, exit 0.
%s

# Best-effort reentrancy guard (no env vars).
REPO_DIR="$(pwd)"
HOOK_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" 2>/dev/null && pwd)"
LOCK_DIR="$HOOK_DIR/.sage-%s.lock"
if ! mkdir "$LOCK_DIR" 2>/dev/null; then
	exit 0
fi
trap 'rmdir "$LOCK_DIR" 2>/dev/null || true' EXIT

if command -v sage >/dev/null 2>&1; then
	%s
fi

# Chain legacy hook if present (best-effort).
if [ -n "${LEGACY_HOOK:-}" ] && [ -x "${LEGACY_HOOK}" ]; then
	"${LEGACY_HOOK}" "$@" || true
fi

exit 0
`, hookName, legacyLine, hookName, sageInvoke)) + "\n"
}

func escapeForDoubleQuotes(s string) string {
	// Keep this intentionally minimal; hook paths are file paths.
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
