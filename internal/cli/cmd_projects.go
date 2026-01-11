package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type shellKind string

const (
	shellSh   shellKind = "sh"
	shellFish shellKind = "fish"
)

var projectsShell string
var projectsRepo string

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage project scope (optional)",
	Long: "Sage stores all entries in one global database (~/.sage/sage.db).\n\n" +
		"Projects are an optional scope: when activated in your shell, `add`, `timeline`,\n" +
		"`state`, and `tag` default to that project. `view <id>` always stays global.\n\n" +
		"Activation prints shell code you should eval (like a Python venv).",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProjectsList()
	},
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List known projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runProjectsList()
	},
}

var projectsCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current active project (if any)",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := activeProjectFromEnv()
		if p == "" {
			fmt.Println("(none)")
			return nil
		}
		fmt.Println(p)
		return nil
	},
}

var projectsDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Suggest a project name from the current repo",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := suggestedProjectFromRepo(projectsRepo)
		if s == "" {
			fmt.Println("(unknown)")
			return nil
		}
		fmt.Println(s)
		return nil
	},
}

var projectsActivateCmd = &cobra.Command{
	Use:   "activate <name>",
	Short: "Activate a project scope in your shell",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := normalizeProjectName(args[0])
		if name == "" {
			return fmt.Errorf("invalid project name")
		}

		sk := parseShellKind(projectsShell)
		switch sk {
		case shellFish:
			fmt.Printf("set -gx SAGE_PROJECT %q\n", name)
		default:
			fmt.Printf("export SAGE_PROJECT=%q\n", name)
		}
		return nil
	},
}

var projectsDeactivateCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "Deactivate project scope in your shell",
	RunE: func(cmd *cobra.Command, args []string) error {
		sk := parseShellKind(projectsShell)
		switch sk {
		case shellFish:
			fmt.Println("set -e SAGE_PROJECT")
		default:
			fmt.Println("unset SAGE_PROJECT")
		}
		return nil
	},
}

var projectsPromptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Print the active project (for shell prompts)",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := activeProjectFromEnv()
		if p == "" {
			return nil
		}
		fmt.Printf("sage:%s", p)
		return nil
	},
}

func runProjectsList() error {
	s, err := openGlobalStore()
	if err != nil {
		return err
	}

	projects, err := s.ListProjects()
	if err != nil {
		return err
	}

	// Hide the implicit default.
	filtered := make([]string, 0, len(projects))
	for _, p := range projects {
		p = normalizeProjectName(p)
		if p == "" || p == defaultProjectName {
			continue
		}
		filtered = append(filtered, p)
	}
	sort.Strings(filtered)

	cur := activeProjectFromEnv()
	if cur == "" {
		cur = "(none)"
	}

	fmt.Printf("Active: %s\n\n", cur)
	fmt.Println("Known projects:")
	if len(filtered) == 0 {
		fmt.Println("(none yet)")
	} else {
		for _, p := range filtered {
			mark := " "
			if cur != "(none)" && p == cur {
				mark = "*"
			}
			fmt.Printf("%s %s\n", mark, p)
		}
	}

	fmt.Println()
	fmt.Println("Activate (bash/zsh): eval \"$(sage projects activate <name>)\"")
	fmt.Println("Activate (fish):     sage projects activate <name> --shell fish | source")
	fmt.Println("Deactivate:          eval \"$(sage projects deactivate)\"")
	return nil
}

func parseShellKind(s string) shellKind {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, ".")
	switch s {
	case "fish":
		return shellFish
	default:
		return shellSh
	}
}

func init() {
	projectsCmd.PersistentFlags().StringVar(&projectsShell, "shell", "sh", "shell type for activation (sh|fish)")
	projectsCmd.PersistentFlags().StringVar(&projectsRepo, "repo", "", "path to repo (for detect)")
	_ = projectsCmd.PersistentFlags().MarkHidden("repo")

	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsCurrentCmd)
	projectsCmd.AddCommand(projectsDetectCmd)
	projectsCmd.AddCommand(projectsActivateCmd)
	projectsCmd.AddCommand(projectsDeactivateCmd)
	projectsCmd.AddCommand(projectsPromptCmd)

	rootCmd.AddCommand(projectsCmd)
}
