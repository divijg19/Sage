# `Sage` (_Chronicle_)

> **A local-first, event-sourced developer cognition engine for capturing decisions, reasoning, and meaning over time.**

`Sage` captures not just _what_ you did—but **why** you did it.

It is a time-aware personal system for developers who want to preserve context, reasoning, and conceptual understanding over time.

---

## Why `Sage`?

Code remembers _changes_.  
Task managers remember _intent_.  
Logs remember _events_.

**None remember reasoning.**

After weeks or months, developers inevitably ask:

- _Why did we choose this approach?_
- _What problem was this solving?_
- _What alternatives were rejected?_
- _What changed my mind?_

`Sage` exists to answer those questions—**locally, permanently, and without friction**.

---

## What Is `Sage`?

`Sage` is a **local-first CLI developer cognition engine** that records work as immutable events.

The CLI is the source of truth today. More derived “cognition layers” (graphs, relationships, projections) are planned later, but the event log remains primary.

Think:

> **Git + journaling + event sourcing + semantic context — for humans.**

---

## Core Principles

- **Local-first** — No cloud, no accounts, no telemetry
- **Append-only** — History is immutable
- **Time-aware** — Past state can be reconstructed
- **Derived meaning** — Graphs are projections, not truth
- **Low-friction** — Designed for daily use
- **Human-centric** — Built for reasoning, not metrics

---

## Current CLI (v0.3)

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

### Tags

Tags are optional strings used for filtering and finding entries.

```bash
# List tags + counts
sage tag

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
```

---

## Key Capabilities

### 🧾 Event-Sourced Timeline (Source of Truth)

All entries are stored as events (entry content is append-only; tags are optional metadata for filtering):

- decisions
- notes
- experiments
- reflections
- outcomes

Today, the CLI focuses on reliable capture and trustworthy summaries.

Every event is timestamped, durable, and replayable.

### 🧠 Derived Cognition (Planned)

From the event log, `Sage` can eventually build a **local semantic graph** of:

- **Concepts** (e.g. `auth`, `postgres`, `event-sourcing`)
- **Decisions** (explicit architectural or technical choices)
- **Artifacts** (modules, files, repos, docs)
- **Relationships** (`affects`, `depends_on`, `supersedes`, `references`)

> **Events are the source of truth.  
> The graph is a projection.**

This ensures:

- explainability
- reversibility
- time-aware reasoning
- zero hallucination

---

### ⚡ `Sage` CLI

Minimal, editor-first commands:

```bash
sage add "Refactored auth middleware" --tags auth,backend
sage add d "Use Go instead of Node for websocket server" --tags backend
sage timeline
sage timeline --tags auth
sage state --at 2026-01-09
```

---

### 🧠 Time-Travel Introspection

Ask questions of the past:

```bash
sage state --at 2025-01-10
```

`Sage` reconstructs:

- decisions made
- notes taken
- relevant context at that moment

All answers link back to concrete events.

---

### 📁 Global Journal

All entries live in a single, local-only store:

```
~/.sage/
├── sage.db
├── config.json
└── templates/*.md
```

If you have older per-directory stores from previous versions, Sage will import them into the global store the first time the global store is empty.

---

### 🖥️ Optional TUI (Planned)

An interactive timeline built on top of the same event store:

- vertical timeline + graph view
- filter by event or concept
- fuzzy search
- collapsible days
- keyboard-first navigation

---

### 🔗 Git Integration (Planned)

`Sage` can:

- associate events with commits
- run via git hooks
- annotate decisions near code changes

---

## Example Event

```json
{
  "id": "evt_20250302_2141",
  "timestamp": "2025-03-02T21:41:00Z",
  "project": "nargis",
  "kind": "decision",
  "title": "Use Postgres instead of Redis",
  "tags": ["backend", "storage"],
  "content": "Switched from Redis to Postgres due to durability and query needs."
}
```

---

## Architecture Overview

- **Language:** Go
- **Core Model:** Event sourcing (append-only log)
- **Derived Model:** Planned (rebuildable projections)
- **Storage:** SQLite
- **Interfaces:** CLI (default)
- **Scope:** Per-project, local-only

`Sage` intentionally avoids:

- mutable global state
- remote sync
- proprietary formats
- opaque inference

---

## Installation

> `Sage` is currently in active development.

Once released:

```bash
go install github.com/divijg19/sage@latest
```

Or download a prebuilt binary from Releases.

---

## Development Status

| Area              | Status |
| ----------------- | ------ |
| Core event model  | ✅     |
| `sage add`        | ✅     |
| Templates         | ✅     |
| Tags              |        |
| Timeline          | ✅     |
| State (`--at`)    | ✅     |
| Semantic graph    | ⏳     |
| TUI               | ⏳     |
| Git hooks         | ⏳     |

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

### v0.4 — Cognition

- sage view
- tags for organization and filtering

### v0.5 - Developer Ergonomics
- git hooks

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

---

## Philosophy

`Sage` treats **developer cognition** as a first-class artifact.

Code changes, understanding compounds. `Sage` preserves the latter.

Building developer understanding is compounding on your choices, and `Sage` seeks to preserve this.

---

## Non-Goals

`Sage` is **not**:

- a task manager
- a note-taking app
- a Git replacement
- a cloud service

It complements existing tools—it does not replace them.

---

## Contributing

`Sage` is opinionated by design.

Before contributing:

- understand the local-first philosophy
- respect immutability
- avoid feature creep

Open an issue before major changes.

---

Built to integrate into my systems.
