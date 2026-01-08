package cli

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/divijg19/Sage/internal/event"
	"github.com/divijg19/Sage/internal/project"
	"github.com/divijg19/Sage/internal/store"
)

var logCmd = &cobra.Command{
	Use:   "log <message>",
	Short: "Log a development event",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := args[0]

		// 1. Detect project
		projectName, dbPath, err := project.Detect()
		if err != nil {
			return err
		}

		// 2. Open store
		s, err := store.Open(dbPath)
		if err != nil {
			return err
		}

		// 3. Create event
		e := event.Event{
			ID:        uuid.NewString(),
			Timestamp: time.Now(),
			Type:      event.LogEvent,
			Project:   projectName,
			Content:   message,
		}

		// 4. Append event
		if err := s.Append(e); err != nil {
			return err
		}

		fmt.Println("logged:", message)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
