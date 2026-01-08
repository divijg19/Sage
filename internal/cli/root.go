package cli

import (
	"github.com/spf13/cobra"
)

var conceptsFlag []string

var rootCmd = &cobra.Command{
	Use:   "sage",
	Short: "A local-first developer cognition engine",
	Long:  "Sage is a local-first, event-sourced system for capturing developer reasoning over time.",
}

func Execute() error {
	return rootCmd.Execute()
}
