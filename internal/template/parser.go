package template

import "strings"

func parseTemplate(filename, raw string) Template {
	name := strings.TrimSuffix(filename, ".md")
	suggested := ""
	body := raw

	if strings.HasPrefix(raw, "---") {
		parts := strings.SplitN(raw, "---", 3)
		if len(parts) == 3 {
			meta := parts[1]
			body = strings.TrimSpace(parts[2])

			for _, line := range strings.Split(meta, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "suggested_kind:") {
					suggested = strings.TrimSpace(
						strings.TrimPrefix(line, "suggested_kind:"),
					)
				}
			}
		}
	}

	return Template{
		Name:          name,
		SuggestedKind: suggested,
		Body:          body,
	}
}
