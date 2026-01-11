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
		"  sage hooks     Install/manage Git hooks\n" +
		"  sage projects  Activate/list project scope\n" +
		"  sage tag       List tags or tag an entry\n" +
		"  sage timeline  Show timestamp/kind/title summaries\n" +
		"  sage view      View a past entry by numeric ID\n" +
		"  sage state     Reconstruct state at a timestamp\n\n" +
		"Storage: ~/.sage/sage.db (global, local-only).\n" +
		"Editor precedence: ~/.sage/config.json (sage editor) > $SAGE_EDITOR > $EDITOR.",
}

func Execute() error {
	return rootCmd.Execute()
}
