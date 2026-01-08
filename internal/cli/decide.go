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

var decideCmd = &cobra.Command{
	Use:   "decide <message>",
	Short: "Record a decision",
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
			Type:      event.DecideEvent,
			Project:   projectName,
			Content:   message,
			Concepts:  parseConcepts(conceptsFlag),
		}

		if err := s.Append(e); err != nil {
			return err
		}

		fmt.Println("decided:", message)
		return nil
	},
}

func init() {
	decideCmd.Flags().StringSliceVar(
		&conceptsFlag,
		"concepts",
		nil,
		"comma-separated list of concepts",
	)
	rootCmd.AddCommand(decideCmd)
}
