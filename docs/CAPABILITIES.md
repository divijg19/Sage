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

### ⚡ `Sage` CLI

Minimal, editor-first commands:

```bash
sage add "Refactored auth middleware" --tags auth,backend
sage add d "Use Go instead of Node for websocket server" --tags backend
sage timeline
sage timeline --tags auth
sage state --at 2026-01-09
```

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
