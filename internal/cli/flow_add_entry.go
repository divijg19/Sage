package cli

import (
	"fmt"
	"strings"

	"github.com/divijg19/sage/internal/event"
)

//
// Title resolution
//

func resolveTitle(arg string, flag string) (string, error) {
	if arg != "" {
		return arg, nil
	}
	if flag != "" {
		return flag, nil
	}

	title := prompt("Title: ")
	if title == "" {
		return "", fmt.Errorf("title is required")
	}
	return title, nil
}

//
// Entry kind resolution (record vs decision)
//

func resolveKind(
	explicit string,
	suggested string,
) (event.EntryKind, error) {

	if explicit == "decision" || explicit == "d" {
		return event.DecisionKind, nil
	}
	if explicit == "record" || explicit == "r" {
		return event.RecordKind, nil
	}

	if suggested == "decision" {
		if confirmDefaultYes("Template suggests a decision. Save as decision? [Y/n]: ") {
			return event.DecisionKind, nil
		}
		return event.RecordKind, nil
	}

	if confirm("Is this a decision? [y/N]: ") {
		return event.DecisionKind, nil
	}

	return event.RecordKind, nil
}

//
// Tag parsing (future-facing, non-invasive)
//

func parseTags(inputs []string) []string {
	if len(inputs) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var tags []string

	for _, input := range inputs {
		parts := strings.Split(input, ",")
		for _, p := range parts {
			tag := strings.ToLower(strings.TrimSpace(p))
			if tag == "" {
				continue
			}
			if _, exists := seen[tag]; exists {
				continue
			}
			seen[tag] = struct{}{}
			tags = append(tags, tag)
		}
	}

	return tags
}
