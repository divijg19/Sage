# `Sage` (_Chronicle_)

> **A local-first, event-sourced developer cognition engine for capturing decisions, reasoning, and meaning over time.**

`Sage` captures not just _what_ you did—but **why** you did it.

It is a time-aware personal system for developers who want to preserve context, reasoning, and conceptual understanding over time.

Quick install (release binary):

```bash
curl -fsSL https://raw.githubusercontent.com/divijg19/sage/main/install.sh | bash
```

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

Full documentation has been split into the `docs/` directory. For detailed sections, see:

- [Roadmap](ROADMAP.md)
- [CLI & Chronicle](docs/cli.md)
- [Key Capabilities](docs/key_capabilities.md)
- [Global Journal](docs/global_journal.md)
- [Projects](docs/projects.md)
- [Chronicle TUI](docs/chronicle_tui.md)
- [Git Hooks](docs/hooks.md)
- [Example Event](docs/example_event.md)
- [Architecture Overview](docs/architecture.md)

Read the files in `docs/` for the remaining content previously found in this README.

---

## What’s Still Missing (v0.1–v0.5)

Everything in v0.1–v0.5 is usable daily, but there are still a few “sharp edges”
 that should be addressed before calling it truly hardened.

- **Migration safety:** the store migration path should be transactional/fail-lo
ud (and covered by regression tests) so data can’t be dropped on partial failure
s.
- **SQLite robustness:** consider a busy timeout / WAL tuning so concurrent re
aders (timeline/state) + writers (hooks) don’t cause sporadic failures.
- **Non-interactive UX:** commands that prompt should have clear behavior when
 stdin isn’t a TTY (flags-only mode, or a friendly error).
- **Observability:** a lightweight `sage doctor`/healthcheck-style command (or
 status output) would make failures easier to debug.
- **Hooks hardening:** ensure hook execution is resilient across odd repo set
ups (custom hooks path, detached HEAD, unusual `PWD`).

---

## Test Strategy

Sage is deliberately small; the goal is tight tests around correctness and “no s
urprises” UX.

- **Unit tests** (fast, pure helpers)
	- tag parsing/normalization
	- project scope precedence (`SAGE_PROJECT`, `--project`, `--all`)
	- timeline/view formatting rules (IDs, timestamps, tag display)
- **Regression tests** (lock in behaviors that must never change)
	- “no-op” editor exits don’t write entries
	- dedupe against latest entry (scoped by project)
	- hook install/uninstall is idempotent and preserves legacy hooks
	- store migrations are idempotent and stable-order
- **Integration tests** (realistic IO)
	- sqlite open/migrate on temp DB
	- hook script execution in a temp git repo
	- editor invocation via a fake script on PATH

---

## CI

This repo includes a GitHub Actions workflow that runs on every push/PR:

- `gofmt` check
- `go vet ./...`
- `go test ./...`
- `go test -race ./...`

See [.github/workflows/ci.yml](.github/workflows/ci.yml).

---

## Philosophy

`Sage` treats **developer cognition** as a first-class artifact.

Code changes, understanding compounds. `Sage` preserves the latter.

Building developer understanding is compounding on your choices, and `Sage` se
eks to preserve this.

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
