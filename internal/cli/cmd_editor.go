package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var editorCmd = &cobra.Command{
	Use:   "editor [command...]",
	Short: "Set or show Sage's preferred editor",
	Long: "Configure the editor Sage uses when opening entries.\n\n" +
		"By default Sage uses $EDITOR. If you set a preferred editor via this command,\n" +
		"it is stored in ~/.sage/config.json and used across shells.\n\n" +
		"Sage tries to make GUI editors behave predictably by adding common blocking flags\n" +
		"(for example: code/zed -> --wait) when launching the editor.\n\n" +
		"Examples:\n" +
		"  sage editor\n" +
		"  sage editor list\n" +
		"  sage editor code --wait\n" +
		"  sage editor zed --wait\n" +
		"  sage editor vim\n" +
		"  sage editor nano\n" +
		"  sage editor micro\n" +
		"  sage editor --unset",
	DisableFlagParsing: true,
	Args:               cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// With DisableFlagParsing we treat everything after `sage editor` as an argument,
		// so editor commands like `zed --wait` work without needing `--`.
		args = normalizeEditorArgs(args)

		if len(args) == 0 {
			printEditorsWithSelection()
			return nil
		}
		if len(args) == 1 && (args[0] == "list" || args[0] == "--list") {
			printEditorsWithSelection()
			return nil
		}
		if len(args) == 1 && (args[0] == "help" || args[0] == "-h" || args[0] == "--help") {
			return cmd.Help()
		}
		if len(args) == 1 && args[0] == "--unset" {
			if err := unsetConfiguredEditor(); err != nil {
				return err
			}
			fmt.Println("Editor unset.")
			printEditorsWithSelection()
			return nil
		}

		editor := strings.TrimSpace(strings.Join(args, " "))
		if editor == "" {
			return fmt.Errorf("editor command cannot be empty")
		}

		if err := setConfiguredEditor(editor); err != nil {
			return err
		}

		fmt.Println("Editor set to:", editor)
		fmt.Println("Runs as:", effectiveEditorInvocation(editor))
		fmt.Println()
		printEditorsWithSelection()
		return nil
	},
}

type detectedEditor struct {
	bin         string
	recommended string
}

func printEditorsWithSelection() {
	selected, source, cfgPath, err := resolveSelectedEditorForDisplay()
	if err != nil {
		fmt.Println("Editor:", "(error)")
		fmt.Println(err)
		return
	}

	candidates := []detectedEditor{
		{bin: "zed", recommended: "zed --wait"},
		{bin: "code", recommended: "code --wait"},
		{bin: "code-insiders", recommended: "code-insiders --wait"},
		{bin: "codium", recommended: "codium --wait"},
		{bin: "subl", recommended: "subl --wait"},
		{bin: "gedit", recommended: "gedit --wait"},
		{bin: "gnome-text-editor", recommended: "gnome-text-editor --wait"},
		{bin: "kate", recommended: "kate --block"},
		{bin: "micro", recommended: "micro"},
		{bin: "nano", recommended: "nano"},
		{bin: "vim", recommended: "vim"},
		{bin: "nvim", recommended: "nvim"},
	}

	if selected == "" {
		selected = "vi"
		source = "default"
	}

	fmt.Println("Selected editor:")
	if source == "config" {
		fmt.Printf("- %s  (from %s)\n", selected, cfgPath)
	} else {
		fmt.Printf("- %s  (from %s)\n", selected, source)
	}
	fmt.Println("- Runs as:", effectiveEditorInvocation(selected))
	fmt.Println()
	fmt.Println("Detected editors on PATH:")

	selectedBin := ""
	{
		argv := strings.Fields(strings.TrimSpace(selected))
		if len(argv) > 0 {
			selectedBin = argv[0]
		}
	}

	seen := map[string]bool{}
	foundAny := false
	for _, c := range candidates {
		if seen[c.bin] {
			continue
		}
		seen[c.bin] = true
		p, err := exec.LookPath(c.bin)
		if err != nil {
			continue
		}
		foundAny = true
		mark := " "
		if c.bin == selectedBin {
			mark = "*"
		}
		line := fmt.Sprintf("%s %s (%s)", mark, c.bin, p)
		if c.recommended != "" {
			line += "  e.g. 'sage editor " + c.recommended + "'"
		}
		fmt.Println(line)
	}

	if !foundAny {
		fmt.Println("(none of the known editors were found on PATH)")
	}

	// If selection isn't detected, still show that clearly.
	if selectedBin != "" {
		if _, err := exec.LookPath(selectedBin); err != nil {
			fmt.Println()
			fmt.Printf("Note: selected editor binary '%s' was not found on PATH.\n", selectedBin)
		}
	}

	fmt.Println()
	fmt.Println("To change: sage editor <command...>  (example: sage editor zed --wait)")
	fmt.Println("To unset:  sage editor --unset  (falls back to $SAGE_EDITOR/$EDITOR/vi)")
}

func resolveSelectedEditorForDisplay() (selected string, source string, cfgPath string, err error) {
	configured, err := getConfiguredEditor()
	if err != nil {
		return "", "", "", err
	}
	if strings.TrimSpace(configured) != "" {
		p := configPath()
		if p == "" {
			p = "~/.sage/config.json"
		}
		return strings.TrimSpace(configured), "config", p, nil
	}
	if v := strings.TrimSpace(os.Getenv("SAGE_EDITOR")); v != "" {
		return v, "$SAGE_EDITOR", "", nil
	}
	if v := strings.TrimSpace(os.Getenv("EDITOR")); v != "" {
		return v, "$EDITOR", "", nil
	}
	return "vi", "default", "", nil
}

func normalizeEditorArgs(args []string) []string {
	// Allow `sage editor -- <cmd...>` as well, but it's not required.
	if len(args) > 0 && args[0] == "--" {
		return args[1:]
	}
	return args
}

func effectiveEditorInvocation(editor string) string {
	argv := strings.Fields(strings.TrimSpace(editor))
	if len(argv) == 0 {
		return "vi <file>"
	}
	bin := argv[0]
	extra := argv[1:]
	extra = ensureEditorWaitArgs(bin, extra)
	if len(extra) == 0 {
		return bin + " <file>"
	}
	return bin + " " + strings.Join(extra, " ") + " <file>"
}

func init() {
	rootCmd.AddCommand(editorCmd)
}
