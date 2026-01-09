package cli

import (
	"fmt"

	"github.com/divijg19/sage/internal/template"
)

func selectTemplateInteractively(templates []template.Template) *template.Template {
	if len(templates) == 0 {
		return nil
	}

	fmt.Println("Choose template:")
	fmt.Println("  0) empty")
	for i, t := range templates {
		fmt.Printf("  %d) %s\n", i+1, t.Name)
	}
	fmt.Print("> ")

	var choice int
	fmt.Scanln(&choice)

	if choice <= 0 || choice > len(templates) {
		return nil
	}
	return &templates[choice-1]
}
