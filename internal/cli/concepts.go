package cli

import "strings"

func parseConcepts(inputs []string) []string {
	if len(inputs) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var concepts []string

	for _, input := range inputs {
		parts := strings.Split(input, ",")
		for _, p := range parts {
			c := strings.ToLower(strings.TrimSpace(p))
			if c == "" {
				continue
			}
			if _, exists := seen[c]; exists {
				continue
			}
			seen[c] = struct{}{}
			concepts = append(concepts, c)
		}
	}

	return concepts
}
