package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/divijg19/sage/internal/event"
	"github.com/spf13/cobra"
)

var timelineTags []string

var timelineCmd = &cobra.Command{
	Use:   "timeline",
	Short: "Show chronological history of entries",
	Long: "Print a clean chronological view of entries from your global Sage log.\n" +
		"Output is intentionally summary-only (timestamp, kind, title) to avoid noisy content.",
	Example: "  sage timeline\n" +
		"  sage timeline --tags auth\n" +
		"  sage timeline --tags auth,backend",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Open global store
		s, err := openGlobalStore()
		if err != nil {
			return err
		}

		// 2. Read all events
		events, err := s.List()
		if err != nil {
			return err
		}

		// 3. Print events
		want := parseTags(timelineTags)
		for _, e := range events {
			if len(want) > 0 && !eventHasAnyTag(e, want) {
				continue
			}
			printEvent(e)
		}

		return nil
	},
}

func printEvent(e event.Event) {
	ts := e.Timestamp.Format("2006-01-02 15:04")
	kind := string(e.Kind)
	if kind == "" {
		kind = "record"
	}
	title := e.Title
	if title == "" {
		title = "(untitled)"
	}

	tagSuffix := ""
	if len(e.Tags) > 0 {
		copyTags := append([]string(nil), e.Tags...)
		sort.Strings(copyTags)
		for i, t := range copyTags {
			copyTags[i] = "#" + t
		}
		tagSuffix = " " + strings.Join(copyTags, " ")
	}

	fmt.Printf("[%d] [%s] %-8s %s%s\n", e.Seq, ts, kind, title, tagSuffix)
}

func init() {
	timelineCmd.Flags().StringArrayVar(&timelineTags, "tags", nil, "filter by tags (repeatable or comma-separated)")
	rootCmd.AddCommand(timelineCmd)
}

func eventHasAnyTag(e event.Event, want []string) bool {
	if len(want) == 0 {
		return true
	}
	if len(e.Tags) == 0 {
		return false
	}

	set := make(map[string]struct{}, len(e.Tags))
	for _, t := range e.Tags {
		set[strings.ToLower(strings.TrimSpace(t))] = struct{}{}
	}

	for _, w := range want {
		if _, ok := set[w]; ok {
			return true
		}
	}
	return false
}
