## Sage CLI

Sage is intentionally editor-centric and calm.

### Add an entry (2-step flow)

Step 1: provide a title (arg or prompt).  
Step 2: your editor opens with a prefilled template (including a `title:` field).  
Save and close to return to the CLI, then confirm to append the entry.

Notes:

- If you don’t explicitly choose record/decision, Sage will ask.
- Exit the editor without saving to cancel.

Sage also protects you from accidental noise:

- If you close the editor without making meaningful changes, nothing is saved.
- If the content is semantically empty (headings-only / boilerplate-only), nothing is saved.
- If you accidentally repeat the exact same entry (same kind/title/content/tags), nothing is saved.

```bash
# Record
sage add "Investigate flaky CI on linux"

# Add tags (repeatable or comma-separated)
sage add "Fix OAuth callback" --tags auth,backend
sage add "Refactor DB adapter" --tags db --tags cleanup

# Decision (quick shorthand)
sage add d "Use SQLite WAL mode"

# Or via flag
sage add --decision "Switch to Go toolchain 1.23"
```

**Editor setup**

Sage uses a configurable editor command.

Precedence:

1) `sage editor ...` (stored in `~/.sage/config.json`)
2) `$SAGE_EDITOR`
3) `$EDITOR`

Recommended: set it once via `sage editor`.

For VS Code:

```bash
sage editor code --wait
```

If you use a GUI editor, it must **block until the file is closed** (for `code`, that means `--wait`).

Sage will automatically add `--wait` if your editor is VS Code (`code`/`codium`) and you forgot it, but setting it explicitly is recommended.

### Templates

Templates are loaded from:

```text
~/.sage/templates/*.md
```

Use a template by **name**:

```bash
sage add --template decision "Add structured decision notes"
```

Or by **numeric ID** (1-based, sorted by filename; no quotes needed):

```bash
sage add --template 1 "Use template #1"
```

If you prefer selecting interactively:

```bash
sage add --choose-template "Pick a template"
```

Sage automatically strips YAML front matter (like `title:` / `kind:`) from stored content, and it won’t save entries that are unchanged boilerplate or semantically empty.

### Timeline filtering

```bash
sage timeline --tags auth
sage timeline --tags auth,backend
sage timeline --all
sage timeline --project myapp
```

Timeline output includes a **numeric entry ID** (the first bracket). Use it with:

```bash
sage view 42
```

### View past entries

```bash
sage view 42
```

This prints the full entry (timestamp, kind, title, tags, content).

If the entry belongs to a project, `sage view` prints `Project: <name>`.

### Tags

Tags are optional strings used for filtering and finding entries.

```bash
# List tags + counts
sage tag

# Scope by project (optional)
sage tag --project myapp
sage tag --all

# Tag an entry id with a tag (comma-separated supported)
sage tag 42 "auth"
sage tag 42 "auth,backend"

# Show all entries with a tag
sage tag "auth"
```

### State reconstruction

```bash
sage state --at 2026-01-09
sage state --at 2026-01-09T21:30
sage state --at 2026-01-09 --project myapp
sage state --at 2026-01-09 --all
```
