package cli

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/divijg19/sage/internal/event"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag [id] [name]",
	Short: "List tags, or tag entries",
	Long: "Tags are optional, user-defined strings used for filtering and finding entries.\n\n" +
		"Forms:\n" +
		"  sage tag                 List configured tags with counts (global)\n" +
		"  sage tag \"name\"          List entries with tag (global)\n" +
		"  sage tag <id> \"name\"     Apply tag(s) to an entry (comma-separated supported)",
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := openGlobalStore()
		if err != nil {
			return err
		}

		switch len(args) {
		case 0:
			return runTagList(s)
		case 1:
			name := strings.TrimSpace(args[0])
			if name == "" {
				return runTagList(s)
			}
			if isDigitsOnly(name) {
				return fmt.Errorf("to tag an entry, use: sage tag <id> \"name\"")
			}
			name = strings.TrimPrefix(name, "#")
			return runTagShow(s, name)
		case 2:
			id, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
			if err != nil || id <= 0 {
				return fmt.Errorf("invalid entry id: %s", args[0])
			}
			return runTagApply(s, id, args[1])
		default:
			return fmt.Errorf("usage: sage tag | sage tag \"name\" | sage tag <id> \"name\"")
		}
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
}

func runTagList(s storeLike) error {
	events, err := s.List()
	if err != nil {
		return err
	}

	counts := make(map[string]int)
	seenInEntries := make(map[string]struct{})
	for _, e := range events {
		for _, t := range parseTags(e.Tags) {
			counts[t]++
			seenInEntries[t] = struct{}{}
		}
	}

	configured, err := getConfiguredTags()
	if err != nil {
		return err
	}

	// Keep configured ordering, but also include any tags found in entries.
	cfgSet := make(map[string]struct{}, len(configured))
	for _, t := range configured {
		cfgSet[t] = struct{}{}
	}
	for t := range seenInEntries {
		if _, ok := cfgSet[t]; ok {
			continue
		}
		configured = append(configured, t)
		cfgSet[t] = struct{}{}
	}
	configured = parseTags(configured)

	// Persist union so tags are truly “global + discoverable”.
	if err := setConfiguredTags(configured); err != nil {
		return err
	}

	fmt.Println("Tags:")
	if len(configured) == 0 {
		fmt.Println("(none)")
	} else {
		for _, t := range configured {
			fmt.Printf("- #%s (%d)\n", t, counts[t])
		}
	}

	fmt.Println()
	fmt.Println("To apply: sage tag <id> \"name\"  (comma-separated supported)")
	fmt.Println("To view:  sage tag \"name\"")

	if !stdinIsTTY() {
		return nil
	}

	name := strings.TrimSpace(prompt("New tag name (blank to exit): "))
	if name == "" {
		return nil
	}

	tags := parseTags([]string{name})
	if len(tags) == 0 {
		return fmt.Errorf("invalid tag")
	}
	if err := ensureTagsConfigured(tags); err != nil {
		return err
	}
	fmt.Println("tag added:", "#"+tags[0])
	return nil
}

func runTagShow(s storeLike, name string) error {
	tags := parseTags([]string{name})
	if len(tags) == 0 {
		return fmt.Errorf("invalid tag")
	}
	want := tags[0]

	events, err := s.List()
	if err != nil {
		return err
	}

	fmt.Printf("Entries tagged '%s':\n", want)
	found := false
	for _, e := range events {
		if eventHasAnyTag(e, []string{want}) {
			printEvent(e)
			found = true
		}
	}
	if !found {
		fmt.Println("(none)")
	}
	return nil
}

func runTagApply(s storeTagger, id int64, rawName string) error {
	tags := parseTags([]string{rawName})
	if len(tags) == 0 {
		return fmt.Errorf("invalid tag")
	}

	e, err := s.GetBySeq(id)
	if err != nil {
		return err
	}
	if e == nil {
		return fmt.Errorf("no entry with id %d", id)
	}

	current := parseTags(e.Tags)
	set := make(map[string]struct{}, len(current))
	merged := append([]string(nil), current...)
	for _, t := range current {
		set[t] = struct{}{}
	}
	for _, t := range tags {
		if _, ok := set[t]; ok {
			continue
		}
		merged = append(merged, t)
		set[t] = struct{}{}
	}

	// Stable output ordering.
	outTags := append([]string(nil), tags...)
	sort.Strings(outTags)

	if err := ensureTagsConfigured(merged); err != nil {
		return err
	}
	if err := s.UpdateTagsBySeq(id, merged); err != nil {
		return err
	}

	fmt.Printf("Tagged entry %d with %s\n", id, formatTags(outTags))
	return nil
}

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return "(none)"
	}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, "#"+t)
	}
	return strings.Join(out, " ")
}

func isDigitsOnly(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func stdinIsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Small interfaces to keep cmd_tag testable and avoid importing store in this file.
// In practice we pass *store.Store.

type storeLike interface {
	List() ([]event.Event, error)
}

type storeTagger interface {
	storeLike
	GetBySeq(seq int64) (*event.Event, error)
	UpdateTagsBySeq(seq int64, tags []string) error
}
