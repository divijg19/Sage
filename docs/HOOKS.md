### 🔗 Git Hooks

Sage can install a safe `post-commit` hook that records a lightweight commit event into your global log.

```bash
# Install into the current repo (backs up/chains existing hooks)
sage hooks install

# Check status
sage hooks status

# Uninstall (restores legacy hook if it was backed up)
sage hooks uninstall
```

Hook behavior:

- Never blocks commits (best-effort, always exits 0)
- Default is non-blocking background execution
- If you prefer synchronous execution: `sage hooks install --sync`