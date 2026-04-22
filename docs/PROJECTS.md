### 📦 Projects (Optional)

Sage defaults to a global view across all entries. Projects are an optional scope you can activate in your shell (like a Python venv):

```bash
# bash/zsh
eval "$(sage projects activate myapp)"

# fish
sage projects activate myapp --shell fish | source
```

When a project is active, these commands default to that scope:

- `sage add`
- `sage timeline`
- `sage state`
- `sage tag` (listing/showing)

`sage view <id>` is always global by numeric ID.

To see your current scope:

```bash
sage projects current
```
