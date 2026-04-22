package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/divijg19/sage/internal/entryflow"
)

type editorLaunch struct {
	cmd      *exec.Cmd
	tempPath string
}

func prepareEditorLaunch(template string) (*editorLaunch, error) {
	editor, err := resolveEditorCommand()
	if err != nil {
		return nil, err
	}
	if editor == "" {
		editor = "vi"
	}

	argv := strings.Fields(editor)
	if len(argv) == 0 {
		argv = []string{"vi"}
	}
	bin := argv[0]
	extraArgs := argv[1:]

	extraArgs = ensureEditorWaitArgs(bin, extraArgs)

	tmpFile, err := os.CreateTemp("", "sage-*.md")
	if err != nil {
		return nil, err
	}

	if _, err := tmpFile.WriteString(template); err != nil {
		_ = os.Remove(tmpFile.Name())
		return nil, err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return nil, err
	}

	cmd := exec.Command(bin, append(extraArgs, tmpFile.Name())...)

	return &editorLaunch{
		cmd:      cmd,
		tempPath: tmpFile.Name(),
	}, nil
}

func (l *editorLaunch) command() *exec.Cmd {
	if l == nil {
		return nil
	}
	return l.cmd
}

func (l *editorLaunch) result() (string, error) {
	if l == nil {
		return "", fmt.Errorf("editor launch is required")
	}

	content, err := os.ReadFile(l.tempPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (l *editorLaunch) cleanup() {
	if l == nil {
		return
	}
	_ = os.Remove(l.tempPath)
}

// openEditor opens the user's editor with a template
// and returns the resulting content verbatim.
// If the editor is aborted (Ctrl+C), it returns an empty string and nil error.
func openEditor(template string) (string, error) {
	launch, err := prepareEditorLaunch(template)
	if err != nil {
		return "", err
	}
	defer launch.cleanup()

	cmd := launch.command()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", nil
		}
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("editor not found: %s", cmd.Path)
		}
		return "", err
	}

	return launch.result()
}

func resolveEditorCommand() (string, error) {
	configured, err := getConfiguredEditor()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(configured) != "" {
		return configured, nil
	}
	if v := strings.TrimSpace(os.Getenv("SAGE_EDITOR")); v != "" {
		return v, nil
	}
	return strings.TrimSpace(os.Getenv("EDITOR")), nil
}

func hasWaitFlag(args []string) bool {
	for _, a := range args {
		switch a {
		case "--wait", "-w":
			return true
		}
	}
	return false
}

func hasAnyFlag(args []string, flags ...string) bool {
	if len(args) == 0 || len(flags) == 0 {
		return false
	}
	for _, a := range args {
		for _, f := range flags {
			if a == f {
				return true
			}
		}
	}
	return false
}

func ensureEditorWaitArgs(bin string, extraArgs []string) []string {
	// Some GUI editors return immediately unless told to wait.
	switch bin {
	case "code", "code-insiders", "codium", "zed", "subl", "gedit", "gnome-text-editor":
		if !hasWaitFlag(extraArgs) {
			return append(extraArgs, "--wait")
		}
		return extraArgs
	case "kate":
		if !hasAnyFlag(extraArgs, "--block") {
			return append(extraArgs, "--block")
		}
		return extraArgs
	default:
		return extraArgs
	}
}

func prepareEditorBody(tpl string, title string) string {
	return entryflow.PrepareEditorBody(tpl, title)
}

func resolveEditorKindSeed(explicitKind string, suggested string) string {
	return entryflow.ResolveEditorKindSeed(explicitKind, suggested)
}

func ensureFrontMatter(body string, title string, kind string) string {
	return entryflow.EnsureFrontMatter(body, title, kind)
}

func defaultEditorTemplate(explicitKind string) string {
	return entryflow.DefaultEditorTemplate(explicitKind)
}

// extractTitleAndBodyFromEditor looks for YAML front matter at the top of the
// buffer and extracts a `title:` field (if present). It returns the extracted
// title (possibly empty) and the remaining body.
func extractMetaAndBodyFromEditor(raw string) (string, string, string) {
	return entryflow.ExtractMetaAndBodyFromEditor(raw)
}

func stripBoilerplate(s string) string {
	return entryflow.StripBoilerplate(s)
}

func normalizeForComparison(raw string) string {
	return entryflow.NormalizeForComparison(raw)
}

func normalizePlainText(s string) string {
	return entryflow.NormalizePlainText(s)
}

// isMeaningfulContent is intentionally conservative: it requires at least one
// non-heading line containing a letter or number.
func isMeaningfulContent(body string) bool {
	return entryflow.IsMeaningfulContent(body)
}
