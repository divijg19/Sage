package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/divijg19/sage/internal/event"
)

var stateAt string
var stateTags []string

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Reconstruct state at a point in time",
	Long: "Replays the event log up to a given timestamp and prints a concise view\n" +
		"of decisions and contextual records.\n\n" +
		"Use --at with RFC3339, local datetime (YYYY-MM-DDTHH:MM), or date-only (YYYY-MM-DD).",
	Example: "  sage state --at 2026-01-09\n" +
		"  sage state --at 2026-01-09T21:30\n" +
		"  sage state --at 2026-01-09T23:59:59+05:30",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Parse timestamp
		t, err := parseTime(stateAt)
		if err != nil {
			return fmt.Errorf("invalid time format, use RFC3339 or YYYY-MM-DD")
		}

		// 2. Open global store
		s, err := openGlobalStore()
		if err != nil {
			return err
		}

		// 3. Load events up to time
		events, err := s.ListUntil(t)
		if err != nil {
			return err
		}

		// Optional tag filter.
		want := parseTags(stateTags)
		if len(want) > 0 {
			filtered := make([]event.Event, 0, len(events))
			for _, e := range events {
				if eventHasAnyTag(e, want) {
					filtered = append(filtered, e)
				}
			}
			events = filtered
		}

		// 5. Replay & print
		replayState(events, t)
		return nil
	},
}

func replayState(events []event.Event, at time.Time) {
	fmt.Printf("State at %s\n\n", at.Format(time.RFC3339))

	fmt.Println("Decisions:")
	for _, e := range events {
		if e.Kind == event.DecisionKind {
			title := strings.TrimSpace(e.Title)
			if title == "" {
				title = "(untitled)"
			}
			fmt.Printf("- [%d] %s\n", e.Seq, title)
		}
	}

	fmt.Println("\nContext:")
	for _, e := range events {
		if e.Kind == event.RecordKind {
			title := strings.TrimSpace(e.Title)
			if title == "" {
				title = "(untitled)"
			}
			fmt.Printf("- [%d] %s\n", e.Seq, title)
		}
	}
}

func parseTime(input string) (time.Time, error) {
	// 1. Full RFC3339
	if t, err := time.Parse(time.RFC3339, input); err == nil {
		return t, nil
	}

	// 2. Local datetime
	if t, err := time.ParseInLocation(
		"2006-01-02T15:04",
		input,
		time.Local,
	); err == nil {
		return t, nil
	}

	// 3. Date only
	if t, err := time.ParseInLocation(
		"2006-01-02",
		input,
		time.Local,
	); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time format")
}

func init() {
	stateCmd.Flags().StringVar(
		&stateAt,
		"at",
		"",
		"timestamp (RFC3339 or YYYY-MM-DD)",
	)
	stateCmd.Flags().StringArrayVar(&stateTags, "tags", nil, "filter replay by tags (repeatable or comma-separated)")
	stateCmd.MarkFlagRequired("at")
	rootCmd.AddCommand(stateCmd)
}
