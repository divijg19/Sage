package cli

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/divijg19/sage/internal/event"
	"github.com/divijg19/sage/internal/project"
	"github.com/divijg19/sage/internal/store"
)

var noteCmd = &cobra.Command{
	Use:   "note <message>",
	Short: "Record a reflective note",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := args[0]

		projectName, dbPath, err := project.Detect()
		if err != nil {
			return err
		}

		s, err := store.Open(dbPath)
		if err != nil {
			return err
		}

		e := event.Event{
			ID:        uuid.NewString(),
			Timestamp: time.Now(),
			Type:      event.NoteEvent,
			Project:   projectName,
			Content:   message,
		}

		if err := s.Append(e); err != nil {
			return err
		}

		fmt.Println("noted:", message)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(noteCmd)
}
