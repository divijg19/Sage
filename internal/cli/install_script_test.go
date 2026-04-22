package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallScript_InstallsWithoutAliasByDefault(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("installer only targets unix platforms")
	}

	repoRoot := repoRootFromPackage(t)
	releaseRoot := t.TempDir()
	version := "v0.0.1"
	assetDir := filepath.Join(releaseRoot, "releases", "download", version)
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	assetName := installerAssetName(version)
	if err := writeTestTarball(filepath.Join(assetDir, assetName)); err != nil {
		t.Fatalf("writeTestTarball: %v", err)
	}

	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	rcFile := filepath.Join(home, ".bashrc")

	cmd := exec.Command("bash", filepath.Join(repoRoot, "install.sh"), "--version", version, "--bin-dir", binDir)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"SAGE_INSTALL_BASE_URL=file://"+filepath.Join(releaseRoot, "releases", "download"),
		"SAGE_INSTALL_RC_FILE="+rcFile,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(filepath.Join(binDir, "sage")); err != nil {
		t.Fatalf("expected sage binary: %v", err)
	}

	rcContents, err := os.ReadFile(rcFile)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ReadFile rc: %v", err)
	}
	if strings.Contains(string(rcContents), "chronicle()") {
		t.Fatalf("did not expect chronicle alias block without opt-in")
	}
}

func TestInstallScript_ChronicleAliasIsIdempotent(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("installer only targets unix platforms")
	}

	repoRoot := repoRootFromPackage(t)
	releaseRoot := t.TempDir()
	version := "v0.0.2"
	assetDir := filepath.Join(releaseRoot, "releases", "download", version)
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	assetName := installerAssetName(version)
	if err := writeTestTarball(filepath.Join(assetDir, assetName)); err != nil {
		t.Fatalf("writeTestTarball: %v", err)
	}

	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	rcFile := filepath.Join(home, ".bashrc")

	for i := 0; i < 2; i++ {
		cmd := exec.Command("bash", filepath.Join(repoRoot, "install.sh"),
			"--version", version,
			"--bin-dir", binDir,
			"--alias",
			"--shell", "bash",
		)
		cmd.Env = append(os.Environ(),
			"HOME="+home,
			"SAGE_INSTALL_BASE_URL=file://"+filepath.Join(releaseRoot, "releases", "download"),
			"SAGE_INSTALL_RC_FILE="+rcFile,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("install.sh run %d failed: %v\n%s", i+1, err, out)
		}
	}

	rcContents, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("ReadFile rc: %v", err)
	}
	content := string(rcContents)
	if strings.Count(content, "# >>> sage chronicle alias >>>") != 1 {
		t.Fatalf("expected a single alias block, got:\n%s", content)
	}
	if !strings.Contains(content, "chronicle()") || !strings.Contains(content, `sage tui "$@"`) {
		t.Fatalf("expected chronicle shell function, got:\n%s", content)
	}
}

func TestInstallScript_LegacyChronicleAliasFlagRejected(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("installer only targets unix platforms")
	}

	repoRoot := repoRootFromPackage(t)
	releaseRoot := t.TempDir()
	version := "v0.0.3"
	assetDir := filepath.Join(releaseRoot, "releases", "download", version)
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	assetName := installerAssetName(version)
	if err := writeTestTarball(filepath.Join(assetDir, assetName)); err != nil {
		t.Fatalf("writeTestTarball: %v", err)
	}

	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	rcFile := filepath.Join(home, ".bashrc")

	cmd := exec.Command("bash", filepath.Join(repoRoot, "install.sh"),
		"--version", version,
		"--bin-dir", binDir,
		"--enable-chronicle-alias",
		"--shell", "bash",
	)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"SAGE_INSTALL_BASE_URL=file://"+filepath.Join(releaseRoot, "releases", "download"),
		"SAGE_INSTALL_RC_FILE="+rcFile,
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected install.sh to reject legacy alias flag, got success\n%s", out)
	}
	if !strings.Contains(string(out), "unknown argument: --enable-chronicle-alias") {
		t.Fatalf("expected unknown-argument error for legacy alias flag, got:\n%s", out)
	}
}

func repoRootFromPackage(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func installerAssetName(version string) string {
	goos := runtime.GOOS
	if goos == "darwin" {
		return "sage_" + version + "_darwin_" + installerArch() + ".tar.gz"
	}
	return "sage_" + version + "_linux_" + installerArch() + ".tar.gz"
}

func installerArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return runtime.GOARCH
	}
}

func writeTestTarball(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	body := []byte("#!/usr/bin/env bash\necho sage\n")
	hdr := &tar.Header{
		Name: "sage",
		Mode: 0o755,
		Size: int64(len(body)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = tw.Write(body)
	return err
}

func TestChronicleAliasBlockTemplate(t *testing.T) {
	scriptPath := filepath.Join(repoRootFromPackage(t), "install.sh")
	cmd := exec.Command("bash", "-lc", "source "+shellEscape(scriptPath)+"; chronicle_alias_block")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("chronicle_alias_block failed: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte(`sage tui "$@"`)) {
		t.Fatalf("expected alias body to route to sage tui, got:\n%s", out)
	}
}

func shellEscape(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}
