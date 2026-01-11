package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/divijg19/sage/internal/event"
	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View a past entry by numeric ID",
	Long: "View the full contents of a past entry using its numeric ID (shown in `sage timeline`).\n\n" +
		"Entries live in a single global log; IDs are global and numeric.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
		if err != nil || id <= 0 {
			return fmt.Errorf("invalid entry id: %s", args[0])
		}

		s, err := openGlobalStore()
		if err != nil {
			return err
		}

		e, err := s.GetBySeq(id)
		if err != nil {
			return err
		}
		if e == nil {
			return fmt.Errorf("no entry with id %d", id)
		}

		printFullEntry(*e)
		return nil
	},
}

func printFullEntry(e event.Event) {
	fmt.Printf("ID: %d\n", e.Seq)
	fmt.Printf("When: %s\n", e.Timestamp.Format("2006-01-02 15:04:05"))
	kind := string(e.Kind)
	if kind == "" {
		kind = "record"
	}
	fmt.Printf("Kind: %s\n", kind)

	title := strings.TrimSpace(e.Title)
	if title == "" {
		title = "(untitled)"
	}
	fmt.Printf("Title: %s\n", title)

	if len(e.Tags) == 0 {
		fmt.Println("Tags: (none)")
	} else {
		copyTags := append([]string(nil), e.Tags...)
		sort.Strings(copyTags)
		for i, t := range copyTags {
			copyTags[i] = "#" + t
		}
		fmt.Printf("Tags: %s\n", strings.Join(copyTags, " "))
	}

	fmt.Println()
	fmt.Println(strings.TrimRight(e.Content, "\n"))
}

func init() {
	rootCmd.AddCommand(viewCmd)
}
