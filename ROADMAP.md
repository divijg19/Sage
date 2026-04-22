# ROADMAP

## Development Status

| Area              | Status |
| ----------------- | ------ |
| Core event model  | ✅     |
| `sage add`        | ✅     |
| Templates         | ✅     |
| Tags              | ✅     |
| Timeline          | ✅     |
| State (`--at`)    | ✅     |
| Semantic graph    | ⏳     |
| TUI               | ✅     |
| Git hooks         | ✅     |
| Projects (scope)  | ✅     |

---

## Roadmap

### v0.1 — Foundation ✅

- append-only event store
- CLI logging & querying
- project scoping
- timeline & time-travel (`state --at`)

### v0.2 — Structure ✅

- stabilize storage + project scoping

### v0.3 — Stabilization ✅

- hardened editor-centric `add` flow (no empty/noisy entries)
- templates by name or numeric id
- clean, trustworthy timeline summaries
- editor config

### v0.4 — Cognition ✅

- sage view
- tags for organization and filtering

### v0.5 - Developer Ergonomics ✅
- git hooks 
- projects (optional scope) 

### v0.6 — UX

- TUI (timeline + graph views)
- fuzzy search
- interactive filtering
- semantic graph projection (derived from events)
- derived projections from concept → decision → artifact relationships

### v0.6+ — Augmentation (Optional)

- local AI summarization
- decision embeddings
- knowledge graph export
