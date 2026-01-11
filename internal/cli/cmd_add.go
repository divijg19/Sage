package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/divijg19/sage/internal/event"
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

		body := ""
		suggested := ""
		seedKind := resolveEditorKindSeed(explicitKind, suggested)

		if chosen != nil {
			prepared := prepareEditorBody(chosen.Body, title)
			suggested = chosen.SuggestedKind
			seedKind = resolveEditorKindSeed(explicitKind, suggested)
			body = ensureFrontMatter(prepared, title, seedKind)
		} else {
			seedKind = resolveEditorKindSeed(explicitKind, suggested)
			body = ensureFrontMatter(
				prepareEditorBody(defaultEditorTemplate(explicitKind), title),
				title,
				seedKind,
			)
		}

		// ---- 5. Editor (unless skipped) ----

		edited, err := openEditor(body)
		if err != nil {
			return err
		}
		if edited == "" {
			return nil
		}

		// If the user didn't change anything from the template, treat it as a no-op.
		if normalizeForComparison(edited) == normalizeForComparison(body) {
			return nil
		}

		editedTitle, editedKind, cleaned := extractMetaAndBodyFromEditor(edited)
		if editedTitle != "" {
			title = editedTitle
		}
		title = strings.TrimSpace(title)
		if title == "" {
			return fmt.Errorf("title is required")
		}

		// If the user edited the kind field, treat it as explicit intent.
		if editedKind != "" && !strings.EqualFold(editedKind, seedKind) {
			explicitKind = editedKind
		}

		content := strings.TrimSpace(cleaned)
		if !isMeaningfulContent(content) {
			return nil
		}

		tags := parseTags(addTags)
		_ = ensureTagsConfigured(tags)

		// ---- 6. Resolve kind (hybrid) ----

		kind, err := resolveKind(explicitKind, suggested)
		if err != nil {
			return err
		}

		if !confirm("Save entry? [y/N]: ") {
			return nil
		}

		// ---- 8. Persist (global) ----

		s, err := openGlobalStore()
		if err != nil {
			return err
		}

		if prev, err := s.Latest(); err == nil && prev != nil {
			if prev.Kind == kind && strings.TrimSpace(prev.Title) == title {
				if normalizePlainText(prev.Content) == normalizePlainText(content) && normalizeTagSet(prev.Tags) == normalizeTagSet(tags) {
					return nil
				}
			}
		}

		e := event.Event{
			ID:        uuid.NewString(),
			Timestamp: time.Now(),
			Project:   "global",
			Kind:      kind,
			Title:     title,
			Content:   content,
			Tags:      tags,
		}

		if err := s.Append(e); err != nil {
			return err
		}

		fmt.Println("entry recorded")
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

func normalizeTagSet(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	// parseTags already lowercases + dedupes; we only need stable ordering.
	copyTags := append([]string(nil), tags...)
	sort.Strings(copyTags)
	return strings.Join(copyTags, ",")
}
