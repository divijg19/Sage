package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/divijg19/sage/internal/event"
	"github.com/divijg19/sage/internal/project"
	"github.com/divijg19/sage/internal/store"
)

var stateAt string

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Reconstruct project state at a point in time",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Parse timestamp
		t, err := time.Parse(time.RFC3339, stateAt)
		if err != nil {
			return fmt.Errorf("invalid time format, use RFC3339")
		}

		// 2. Detect project
		_, dbPath, err := project.Detect()
		if err != nil {
			return err
		}

		// 3. Open store
		s, err := store.Open(dbPath)
		if err != nil {
			return err
		}

		// 4. Load events up to time
		events, err := s.ListUntil(t)
		if err != nil {
			return err
		}

		// 5. Replay & print
		replay(events)
		return nil
},
}

func replay(events []event.Event) {
	fmt.Printf("State at %s\n\n", stateAt)

	for _, e := range events {
		switch e.Type {
		case event.DecideEvent:
			fmt.Printf("DECISION: %s\n", e.Content)
		case event.NoteEvent:
			fmt.Printf("NOTE: %s\n", e.Content)
		case event.LogEvent:
			fmt.Printf("LOG: %s\n", e.Content)
		}
	}
}


func init() {
	stateCmd.Flags().StringVar(
		&stateAt,
		"at",
		"",
		"ISO timestamp (e.g. 2025-01-10T18:30)",
	)

	stateCmd.MarkFlagRequired("at")
	rootCmd.AddCommand(stateCmd)
}
