package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/divijg19/sage/internal/entryflow"
	"github.com/divijg19/sage/internal/template"
)

var (
	addTitle          string
	addTemplate       string
	addDecision       bool
	addChooseTemplate bool
	addTags           []string
)

var addCmd = &cobra.Command{
	Use:   "add [record|r|decision|d] [title]",
	Short: "Add a new entry (record/decision)",
	Long: "Add a new entry to Sage using a calm, editor-centric flow.\n\n" +
		"Flow:\n" +
		"  1) Provide a title (arg or prompt)\n" +
		"  2) Your editor opens with a template (including title/kind front matter)\n" +
		"  3) Save & close to append; exit without saving to cancel\n\n" +
		"Sage will not save entries that are unchanged boilerplate or semantically empty.",
	Example: "  sage add \"Investigate flaky CI on linux\"\n" +
		"  sage add d \"Use SQLite WAL mode\"\n" +
		"  sage add --template 1 \"Template by numeric id\"\n" +
		"  sage add --template decision \"Template by name\"\n" +
		"  sage add \"Fix OAuth callback\" --tags auth,backend --tags cleanup\n" +
		"  EDITOR=\"code --wait\" sage add \"Use VS Code as editor\"",
	RunE: func(cmd *cobra.Command, args []string) error {

		// ---- 1. Explicit kind + title (shorthand) ----

		explicitKind := ""
		titleArg := ""

		if len(args) > 0 {
			switch args[0] {
			case "d", "decision":
				explicitKind = "decision"
				titleArg = strings.TrimSpace(strings.Join(args[1:], " "))
			case "r", "record":
				explicitKind = "record"
				titleArg = strings.TrimSpace(strings.Join(args[1:], " "))
			default:
				titleArg = strings.TrimSpace(strings.Join(args, " "))
			}
		}

		if addDecision {
			explicitKind = "decision"
		}

		// ---- 2. Resolve title ----

		title, err := resolveTitle(titleArg, addTitle)
		if err != nil {
			return err
		}

		// ---- 3. Load templates ----

		templates, _ := template.LoadAll(templateDir())
		var chosen *template.Template

		if addTemplate != "" {
			if id, err := strconv.Atoi(addTemplate); err == nil {
				if id < 0 {
					return fmt.Errorf("invalid template id: %d", id)
				}
				if id == 0 {
					chosen = nil
				} else if id >= 1 && id <= len(templates) {
					t := templates[id-1]
					chosen = &t
				} else {
					return fmt.Errorf("template id out of range: %d", id)
				}
			} else {
				for _, t := range templates {
					if t.Name == addTemplate {
						chosen = &t
						break
					}
				}
				if chosen == nil {
					return fmt.Errorf("template not found: %s", addTemplate)
				}
			}
		} else if addChooseTemplate {
			chosen = selectTemplateInteractively(templates)
		}

		// ---- 4. Prepare editor body ----

		suggested := ""
		if chosen != nil {
			suggested = chosen.SuggestedKind
		}
		templateBody := ""
		if chosen != nil {
			templateBody = chosen.Body
		}
		prepared := entryflow.PrepareInitialBuffer(title, explicitKind, suggested, templateBody)

		// ---- 5. Editor (unless skipped) ----

		edited, err := openEditor(prepared.Body)
		if err != nil {
			return err
		}

		tags := parseTags(addTags)

		// ---- 8. Persist (global DB; project-scoped entries) ----

		s, err := openGlobalStore()
		if err != nil {
			return err
		}

		project := projectForNewEntry()

		result, err := entryflow.Finalize(entryflow.FinalizeRequest{
			Title:         title,
			ExplicitKind:  explicitKind,
			SuggestedKind: suggested,
			SeedKind:      prepared.SeedKind,
			InitialBody:   prepared.Body,
			Edited:        edited,
			Project:       project,
			Tags:          tags,
		}, entryflow.Dependencies{
			Store:       s,
			EnsureTags:  ensureTagsConfigured,
			ResolveKind: resolveKind,
			ConfirmSave: func() bool { return confirm("Save entry? [y/N]: ") },
		})
		if err != nil {
			return err
		}
		if result.Status == entryflow.StatusSaved {
			fmt.Println("entry recorded")
		}
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addTitle, "title", "", "set entry title")
	addCmd.Flags().StringVar(&addTemplate, "template", "", "use template (name or numeric id)")
	addCmd.Flags().BoolVar(&addDecision, "decision", false, "mark as decision")
	addCmd.Flags().BoolVar(&addChooseTemplate, "choose-template", false, "choose a template interactively")
	addCmd.Flags().StringArrayVar(&addTags, "tags", nil, "categorize entry (repeatable or comma-separated, e.g. --tags auth,backend)")

	rootCmd.AddCommand(addCmd)
}
