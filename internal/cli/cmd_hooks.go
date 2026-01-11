package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var hooksRepo string
var hooksHook string
var hooksForce bool
var hooksDryRun bool
var hooksSync bool

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Install and manage Git hooks",
	Long: "Install and manage Sage Git hooks for the current repository (or --repo).\n\n" +
		"Sage hooks are designed to be safe: they never block commits, and will back up\n" +
		"existing hooks and chain them by default.\n\n" +
		"Commit events are recorded under a project derived from the repo name.",
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Sage Git hook(s)",
	RunE: func(cmd *cobra.Command, args []string) error {
		hook, err := validateHookName(hooksHook)
		if err != nil {
			return err
		}

		hooksDir, coreHooksPath, err := gitHooksDir(hooksRepo)
		if err != nil {
			return err
		}

		res, err := InstallHook(hooksDir, hook, HookInstallOptions{Force: hooksForce, DryRun: hooksDryRun, Sync: hooksSync})
		if err != nil {
			return err
		}

		fmt.Printf("Repo hooks dir: %s\n", hooksDir)
		if coreHooksPath != "" {
			fmt.Printf("core.hooksPath: %s\n", coreHooksPath)
			fmt.Println("Warning: core.hooksPath may be shared across repos.")
		}
		fmt.Printf("Installed: %s\n", filepath.Join(hooksDir, hook))
		if res.BackedUp {
			fmt.Printf("Backed up existing hook to: %s\n", res.BackupPath)
		}
		if hooksDryRun {
			fmt.Println("(dry-run) no files were modified")
		}
		return nil
	},
}

var hooksStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show hook installation status",
	RunE: func(cmd *cobra.Command, args []string) error {
		hook, err := validateHookName(hooksHook)
		if err != nil {
			return err
		}

		root, err := gitRepoRoot(hooksRepo)
		if err != nil {
			return err
		}
		hooksDir, coreHooksPath, err := gitHooksDir(hooksRepo)
		if err != nil {
			return err
		}

		ins, err := InspectHook(hooksDir, hook)
		if err != nil {
			return err
		}

		fmt.Printf("Repo: %s\n", root)
		fmt.Printf("Hooks dir: %s\n", hooksDir)
		if coreHooksPath != "" {
			fmt.Printf("core.hooksPath: %s\n", coreHooksPath)
			fmt.Println("Warning: core.hooksPath may be shared across repos.")
		}

		if !ins.Exists {
			fmt.Printf("%s: not installed\n", hook)
			return nil
		}
		fmt.Printf("%s: installed at %s\n", hook, ins.HookPath)
		fmt.Printf("Sage-managed: %t\n", ins.SageManaged)
		if ins.LegacyHookPath != "" {
			fmt.Printf("Chained legacy hook: %s\n", ins.LegacyHookPath)
		}
		return nil
	},
}

var hooksUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Sage Git hook(s)",
	RunE: func(cmd *cobra.Command, args []string) error {
		hook, err := validateHookName(hooksHook)
		if err != nil {
			return err
		}

		hooksDir, coreHooksPath, err := gitHooksDir(hooksRepo)
		if err != nil {
			return err
		}

		msg, err := UninstallHook(hooksDir, hook, HookInstallOptions{Force: hooksForce, DryRun: hooksDryRun})
		if err != nil {
			return err
		}

		fmt.Printf("Repo hooks dir: %s\n", hooksDir)
		if coreHooksPath != "" {
			fmt.Printf("core.hooksPath: %s\n", coreHooksPath)
			fmt.Println("Warning: core.hooksPath may be shared across repos.")
		}
		fmt.Printf("%s: %s\n", hook, msg)
		if hooksDryRun {
			fmt.Println("(dry-run) no files were modified")
		}
		return nil
	},
}

func validateHookName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "post-commit"
	}
	if name != "post-commit" {
		return "", fmt.Errorf("unsupported hook: %s (only post-commit is supported)", name)
	}
	return name, nil
}

func init() {
	hooksCmd.PersistentFlags().StringVar(&hooksRepo, "repo", "", "path to repo (defaults to current directory)")
	hooksCmd.PersistentFlags().StringVar(&hooksHook, "hook", "post-commit", "hook name (currently only post-commit is supported)")
	hooksCmd.PersistentFlags().BoolVar(&hooksForce, "force", false, "overwrite existing hook instead of backing it up")
	hooksCmd.PersistentFlags().BoolVar(&hooksDryRun, "dry-run", false, "print what would change without modifying files")
	hooksInstallCmd.Flags().BoolVar(&hooksSync, "sync", false, "run Sage synchronously on commit (default: background)")

	hooksCmd.AddCommand(hooksInstallCmd)
	hooksCmd.AddCommand(hooksStatusCmd)
	hooksCmd.AddCommand(hooksUninstallCmd)

	rootCmd.AddCommand(hooksCmd)
}
