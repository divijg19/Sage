package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sage",
	Short: "Local-first developer cognition engine",
	Long: "Sage is a local-first, append-only log for capturing developer reasoning over time.\n\n" +
		"Core commands:\n" +
		"  sage add       Add a record/decision (opens your editor)\n" +
		"  sage editor    Configure which editor Sage uses\n" +
		"  sage timeline  Show timestamp/kind/title summaries\n" +
		"  sage state     Reconstruct state at a timestamp\n\n" +
		"Editor precedence: ~/.sage/config.json (sage editor) > $SAGE_EDITOR > $EDITOR.",
}

func Execute() error {
	return rootCmd.Execute()
}
