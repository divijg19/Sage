package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallHook_NewInstall(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "hooks")

	res, err := InstallHook(hooksDir, "post-commit", HookInstallOptions{})
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if res.BackedUp {
		t.Fatalf("expected no backup")
	}

	b, err := os.ReadFile(filepath.Join(hooksDir, "post-commit"))
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if !isSageManagedHook(string(b), "post-commit") {
		t.Fatalf("expected Sage-managed hook")
	}
	if strings.Contains(string(b), "LEGACY_HOOK=\"\"") == false {
		t.Fatalf("expected empty legacy hook path")
	}
	if strings.Contains(string(b), "SAGE_DISABLE_GIT_HOOKS") {
		t.Fatalf("did not expect env-var toggles in hook script")
	}
	if strings.Contains(string(b), "SAGE_HOOK_SYNC") {
		t.Fatalf("did not expect env-var toggles in hook script")
	}
	if !strings.Contains(string(b), "sage hook post-commit") {
		t.Fatalf("expected hook to invoke sage")
	}
	if !strings.Contains(string(b), ") &") {
		t.Fatalf("expected background execution by default")
	}

	st, err := os.Stat(filepath.Join(hooksDir, "post-commit"))
	if err != nil {
		t.Fatalf("stat hook: %v", err)
	}
	if st.Mode()&0o111 == 0 {
		t.Fatalf("expected hook to be executable, mode=%v", st.Mode())
	}
}

func TestInstallHook_SyncMode(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "hooks")

	_, err := InstallHook(hooksDir, "post-commit", HookInstallOptions{Sync: true})
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(hooksDir, "post-commit"))
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	content := string(b)
	if strings.Contains(content, ") &") {
		t.Fatalf("expected synchronous execution (no background '&')")
	}
	if !strings.Contains(content, "sage hook post-commit") {
		t.Fatalf("expected hook to invoke sage")
	}
}

func TestInstallHook_DryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "hooks")

	_, err := InstallHook(hooksDir, "post-commit", HookInstallOptions{DryRun: true})
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if _, err := os.Stat(filepath.Join(hooksDir, "post-commit")); err == nil {
		t.Fatalf("expected hook file to not be created in dry-run")
	}
}

func TestInstallHook_ForceOverwritesWithoutBackup(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	hookPath := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho legacy\n"), 0o755); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	res, err := InstallHook(hooksDir, "post-commit", HookInstallOptions{Force: true})
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if res.BackedUp {
		t.Fatalf("did not expect backup with --force")
	}

	entries, _ := os.ReadDir(hooksDir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".sage.legacy") {
			t.Fatalf("did not expect legacy backup file with --force")
		}
	}
}

func TestUninstallHook_RefusesWhenNotManaged(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	hookPath := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho legacy\n"), 0o755); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	msg, err := UninstallHook(hooksDir, "post-commit", HookInstallOptions{})
	if err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}
	if !strings.Contains(msg, "not Sage-managed") {
		t.Fatalf("expected refusal message, got: %s", msg)
	}
}

func TestInstallHook_BacksUpAndChainsExisting(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	hookPath := filepath.Join(hooksDir, "post-commit")
	legacyContent := "#!/bin/sh\necho legacy\n"
	if err := os.WriteFile(hookPath, []byte(legacyContent), 0o755); err != nil {
		t.Fatalf("write legacy hook: %v", err)
	}

	res, err := InstallHook(hooksDir, "post-commit", HookInstallOptions{})
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if !res.BackedUp {
		t.Fatalf("expected backup")
	}
	if res.BackupPath == "" {
		t.Fatalf("expected backup path")
	}

	b, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if !isSageManagedHook(string(b), "post-commit") {
		t.Fatalf("expected Sage-managed hook")
	}

	ins, err := InspectHook(hooksDir, "post-commit")
	if err != nil {
		t.Fatalf("InspectHook: %v", err)
	}
	if !ins.SageManaged {
		t.Fatalf("expected managed")
	}
	if ins.LegacyHookPath != res.BackupPath {
		t.Fatalf("expected legacy path %q, got %q", res.BackupPath, ins.LegacyHookPath)
	}

	legacyBytes, err := os.ReadFile(res.BackupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(legacyBytes) != legacyContent {
		t.Fatalf("backup content mismatch")
	}
}

func TestInstallHook_IdempotentWithBackup(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	hookPath := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho legacy\n"), 0o755); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	first, err := InstallHook(hooksDir, "post-commit", HookInstallOptions{})
	if err != nil {
		t.Fatalf("first install: %v", err)
	}
	if !first.BackedUp {
		t.Fatalf("expected first install to back up")
	}

	second, err := InstallHook(hooksDir, "post-commit", HookInstallOptions{})
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if second.BackedUp {
		t.Fatalf("expected second install to not back up")
	}

	ins, err := InspectHook(hooksDir, "post-commit")
	if err != nil {
		t.Fatalf("InspectHook: %v", err)
	}
	if ins.LegacyHookPath != first.BackupPath {
		t.Fatalf("expected legacy path preserved %q, got %q", first.BackupPath, ins.LegacyHookPath)
	}
}

func TestUninstallHook_RestoresLegacy(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	hookPath := filepath.Join(hooksDir, "post-commit")
	legacyContent := "#!/bin/sh\necho legacy\n"
	if err := os.WriteFile(hookPath, []byte(legacyContent), 0o755); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	res, err := InstallHook(hooksDir, "post-commit", HookInstallOptions{})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !res.BackedUp {
		t.Fatalf("expected backup")
	}

	msg, err := UninstallHook(hooksDir, "post-commit", HookInstallOptions{})
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if msg != "restored legacy hook" {
		t.Fatalf("unexpected message: %s", msg)
	}

	b, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read restored: %v", err)
	}
	if string(b) != legacyContent {
		t.Fatalf("restored content mismatch")
	}
	if _, err := os.Stat(res.BackupPath); err == nil {
		t.Fatalf("expected backup path to be moved back")
	}
}
