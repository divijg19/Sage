# `Sage` (_Chronicle_)

> **A local-first, event-sourced developer cognition engine for capturing decisions, reasoning, and meaning over time.**

`Sage` captures not just _what_ you did—but **why** you did it.

It is an append-only, time-aware personal system for developers who want to preserve context, reasoning, conceptual understanding and evolution across projects.

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

`Sage` is a **local-first CLI + TUI developer cognition engine** that records work as immutable events and derives a **time-aware semantic graph** of concepts, decisions, and relationships across your code, docs, and projects.

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

## Key Capabilities

### 🧾 Event-Sourced Timeline (Source of Truth)

All interactions are stored as immutable events:

- decisions
- notes
- experiments
- reflections
- outcomes
- commands (optional)

Every event is timestamped, durable, and replayable.

### 🧠 Semantic Graph (Derived Cognition Layer)

From the event log, `Sage` incrementally builds a **local semantic graph** of:

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

Minimal, expressive commands:

```bash
sage log "Refactored auth middleware"
sage decide "Use Go instead of Node for websocket server" --about auth,backend
sage note "Latency dropped after removing Redis"
sage timeline --last 7d
```

---

### 🧠 Time-Travel Introspection

Ask questions of the past:

```bash
sage why auth
sage state --at 2025-01-10
sage impact websocket
```

`Sage` reconstructs:

- decisions made
- notes taken
- concepts involved
- related artifacts
- project context at that moment

All answers link back to concrete events.

---

### 📁 Project-Scoped Journals

`Sage`s are automatically scoped per project:

```
~/.`Sage`/
├── nargis/
├── rig/
├── camellia/
└── global/
```

No manual setup required.

---

### 🖥️ Optional TUI (Terminal UI)

An interactive timeline built with a Go TUI stack:

- vertical timeline + graph view
- filter by event or concept
- fuzzy search
- collapsible days
- keyboard-first navigation

---

### 🔗 Git Integration (Optional)

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
  "type": "decision",
  "project": "nargis",
  "concepts": ["backend", "auth", "storage"],
  "content": "Switched from Redis to Postgres due to durability concerns"
}
```

---

## Architecture Overview

- **Language:** Go
- **Core Model:** Event sourcing (append-only log)
- **Derived Model:** Semantic graph (rebuildable)
- **Storage:** SQLite
- **Interfaces:** CLI (default), TUI (optional)
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
go install github.com/divijg19/`Sage`@latest
```

Or download a prebuilt binary from Releases.

---

## Development Status

| Area              | Status |
| ----------------- | ------ |
| Core event model  | ✅x    |
| CLI logging       | ✅x    |
| Concept tagging   |        |
| Semantic graph    | 🚧     |
| Timeline queries  | 🚧x    |
| Time-travel state | 🚧x    |
| TUI               | ⏳     |
| Git hooks         | ⏳     |

---

## Roadmap

### v0.1 — Foundation [x]

- append-only event store
- CLI logging & querying
- project scoping
- timeline & time-travel (`state --at`)
- log / note / decide commands

### v0.2 — Structure

- markdown export
- structured decisions (`decide`)
- explicit concept tagging
- basic Git integration (commit association)

### v0.3 — Cognition

- semantic graph projection (derived from events)
- concept → decision → artifact relationships
- `sage why`, `sage impact`, `sage trace`

### v0.4 — Ergonomics

- TUI (timeline + graph views)
- fuzzy search
- interactive filtering

### v0.5+ — Augmentation (Optional)

- local AI summarization
- decision embeddings
- knowledge graph export

---

## Philosophy

`Sage` treats **developer cognition** as a first-class artifact.

Code changes, understanding compounds. Building developer understanding is compounding on your choices, and `Sage` seeks to preserve this.

`Sage` preserves the latter.

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
