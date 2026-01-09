package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var stdinReader = bufio.NewReader(os.Stdin)

// prompt displays a label and reads a single line of input.
func prompt(label string) string {
	fmt.Print(label)
	input, _ := stdinReader.ReadString('\n')
	return strings.TrimSpace(input)
}

// confirm asks the user a yes/no question.
// Default is NO unless the user explicitly answers yes.
func confirm(label string) bool {
	fmt.Print(label)
	input, _ := stdinReader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes"
}

// confirmDefaultYes asks the user a yes/no question.
// Default is YES unless the user explicitly answers no.
func confirmDefaultYes(label string) bool {
	fmt.Print(label)
	input, _ := stdinReader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return true
	}
	return !(input == "n" || input == "no")
}
