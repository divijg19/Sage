## 🖥️ Chronicle TUI

```bash
sage tui
```

Chronicle reuses the same global event store and add-flow rules as the rest of the CLI, but presents them as a keyboard-first home screen with:

- an editorial `Chronicle` masthead with scope, result count, and active search/filter summary
- a persistent context rail for scope, active filters, tags, and selected-entry context
- a day-grouped timeline with expandable entries and a dedicated inspector pane
- a dedicated bottom bar that toggles between search and safe in-TUI `sage` commands
- basic search across title, content, tags, project, and kind
- filter controls for project, kind, and tags
- a quick-entry sheet that seeds the note and then opens your configured editor

Layout adapts by terminal width:

- wide terminals show a three-panel layout: context rail, timeline, and inspector
- medium terminals keep the context rail and stack the timeline above the inspector
- compact terminals switch between browse and inspect modes with `Tab`

Useful keys:

- `j` / `k` or arrow keys to move
- `Enter` / `Space` to expand entries or collapse day groups
- `/` to focus the search bar
- `:` to focus the `sage` command bar
- `Tab` to switch search/command modes while the bottom bar is focused
- `f` to open filters
- `n` to start a quick entry
- `r` to reload
- `Tab` to switch between browse and inspect modes on narrow terminals
- `Esc` to close the active drawer/input
- `q` to quit

Command bar shortcuts:

- `add`, `new`, or `record` to open quick entry for a record
- `decision` to open quick entry for a decision
- `filters` to open the filter palette
- `reload` to refresh Chronicle
- `clear` to clear search and filters
- `view <id>` to jump to a visible entry by numeric ID
- `quit` to exit
