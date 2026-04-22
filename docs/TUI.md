## 🖥️ Chronicle TUI

```bash
sage tui
```

Chronicle reuses the same global event store and add-flow rules as the rest of the CLI, but presents them as a keyboard-first home screen with:

- a `Chronicle` header and project-aware scope summary
- a day-grouped timeline with expandable entries
- basic search across title, content, tags, project, and kind
- filter controls for project, kind, and tags
- a quick-entry drawer that seeds the note and then opens your configured editor

Useful keys:

- `j` / `k` or arrow keys to move
- `Enter` / `Space` to expand entries or collapse day groups
- `/` to focus search
- `f` to open filters
- `n` to start a quick entry
- `r` to reload
- `Tab` to toggle the preview drawer on narrow terminals
- `Esc` to close the active drawer/input
- `q` to quit
