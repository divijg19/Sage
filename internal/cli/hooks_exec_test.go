package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstalledHook_ExecutesSageAndLegacyHook(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}

	sageRan := filepath.Join(dir, "sage.ran")
	legacyRan := filepath.Join(dir, "legacy.ran")
	argsLog := filepath.Join(dir, "sage.args")

	// Seed an existing legacy hook.
	legacyHookPath := filepath.Join(hooksDir, "post-commit")
	legacyScript := "#!/bin/sh\n" +
		"touch \"" + legacyRan + "\"\n" +
		"exit 0\n"
	if err := os.WriteFile(legacyHookPath, []byte(legacyScript), 0o755); err != nil {
		t.Fatalf("write legacy hook: %v", err)
	}

	// Install Sage hook (backs up legacy and chains it).
	_, err := InstallHook(hooksDir, "post-commit", HookInstallOptions{})
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	// Provide a fake `sage` binary on PATH.
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	fakeSage := filepath.Join(binDir, "sage")
	fakeSageScript := "#!/bin/sh\n" +
		"echo \"$@\" >> \"" + argsLog + "\"\n" +
		"touch \"" + sageRan + "\"\n" +
		"exit 0\n"
	if err := os.WriteFile(fakeSage, []byte(fakeSageScript), 0o755); err != nil {
		t.Fatalf("write fake sage: %v", err)
	}

	hookPath := filepath.Join(hooksDir, "post-commit")
	cmd := exec.Command("sh", hookPath)
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	cmd.Dir = dir // any dir is fine; hook passes --repo via pwd()

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hook exec failed: %v\n%s", err, string(out))
	}

	if _, err := os.Stat(sageRan); err != nil {
		t.Fatalf("expected fake sage to run: %v", err)
	}
	if _, err := os.Stat(legacyRan); err != nil {
		t.Fatalf("expected legacy hook to run: %v", err)
	}

	b, _ := os.ReadFile(argsLog)
	if !strings.Contains(string(b), "hook post-commit") {
		t.Fatalf("expected sage to be invoked with hook post-commit, got: %s", string(b))
	}
}

func TestInstalledHook_RespectsLockDir(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "hooks")

	_, err := InstallHook(hooksDir, "post-commit", HookInstallOptions{Sync: true})
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	// Fake sage that would create a marker file if invoked.
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	marker := filepath.Join(dir, "sage.ran")
	fakeSage := filepath.Join(binDir, "sage")
	fakeSageScript := "#!/bin/sh\n" +
		"touch \"" + marker + "\"\n" +
		"exit 0\n"
	if err := os.WriteFile(fakeSage, []byte(fakeSageScript), 0o755); err != nil {
		t.Fatalf("write fake sage: %v", err)
	}

	// Pre-create lock dir to simulate an in-flight run.
	lockDir := filepath.Join(hooksDir, ".sage-post-commit.lock")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		t.Fatalf("mkdir lock: %v", err)
	}

	hookPath := filepath.Join(hooksDir, "post-commit")
	cmd := exec.Command("sh", hookPath)
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hook exec failed: %v\n%s", err, string(out))
	}

	if _, err := os.Stat(marker); err == nil {
		t.Fatalf("did not expect sage to run while lock dir exists")
	}
}
