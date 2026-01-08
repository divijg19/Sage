package cli

import (
	"fmt"

	"github.com/divijg19/sage/internal/event"
	"github.com/divijg19/sage/internal/project"
	"github.com/divijg19/sage/internal/store"
	"github.com/spf13/cobra"
)

var timelineCmd = &cobra.Command{
	Use:   "timeline",
	Short: "Show chronological history of events",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Detect project
		_, dbPath, err := project.Detect()
		if err != nil {
			return err
		}

		// 2. Open store
		s, err := store.Open(dbPath)
		if err != nil {
			return err
		}

		// 3. Read all events
		events, err := s.List()
		if err != nil {
			return err
		}

		// 4. Print events
		for _, e := range events {
			printEvent(e)
		}

		return nil
	},
}

func printEvent(e event.Event) {
	ts := e.Timestamp.Format("2006-01-02 15:04")
	fmt.Printf(
		"[%s] %-6s %s\n",
		ts,
		e.Type,
		e.Content,
	)
}

func init() {
	rootCmd.AddCommand(timelineCmd)
}
