## Architecture Overview

- **Language:** Go
- **Core Model:** Event sourcing (append-only log)
- **Derived Model:** Planned (rebuildable projections)
- **Storage:** SQLite
- **Interfaces:** CLI (default)
- **Scope:** Global by default (optional project scope)

### 📁 Global Journal

All entries live in a single, local-only store:

```
~/.sage/
├── sage.db
├── config.json
└── templates/*.md
```

If you have older per-directory stores from previous versions, Sage will import them into the global store the first time the global store is empty.

`Sage` intentionally avoids:

- mutable global state
- remote sync
- proprietary formats
- opaque inference
