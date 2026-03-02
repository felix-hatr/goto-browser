# CLAUDE.md — goto-browser (zebro)

## Project Overview

**zebro** is a macOS terminal URL shortcut manager. Users store URL patterns with variable placeholders and open them in the browser by short key paths.

- **Module**: `github.com/felix-hatr/goto-browser`
- **Binary**: `zebro` (built from `cmd/zebro/main.go`)
- **Stack**: Go + cobra + gopkg.in/yaml.v3 (no viper)
- **Data**: `~/.config/zebro/` (XDG) — global `config.yaml` + `profiles/{name}/` per-profile YAML files

## Build & Test

```bash
# Build
go build -buildvcs=false -o /tmp/zebro ./cmd/zebro

# Test
go test -buildvcs=false ./...

# Install
make install

# Generate shell completions
make completions   # → completions/zebro.{bash,zsh,fish}
```

Always use `-buildvcs=false` — the repo has no git remote configured.

## Project Structure

```
cmd/zebro/         — entry point (sets version via ldflags)
internal/
  cli/             — cobra commands (one file per resource)
    root.go        — rootCmd, global flags, shared helpers
    link.go        — zebro link {create,list,view,rename,delete,clear}
    group.go       — zebro group {create,list,view,add,remove,rename,delete,clear}
    profile.go     — zebro profile {create,list,view,use,rename,delete,backup,restore}
    open.go        — zebro open
    history.go     — zebro history {list,stats,compact,clear}
    config.go      — zebro config {list,get,set}
    doctor.go      — zebro doctor
    backup.go      — backup helpers
    completion.go  — zebro completion
  config/          — config load/save, paths, profile I/O
  store/           — YAML read/write for links/groups; JSONL for history; variable token logic
  resolver/        — URL resolution with scoring
  browser/         — browser open via embedded AppleScript (osascript)
```

## Key Architecture

### Variable Token System

Variables are stored internally as `<vp>name` tokens (prefix `<vp>` is a literal internal marker, not the user's prefix character). At display time, tokens are denormalized back to the user's prefix character.

- `store.NormalizeVars(s, prefix)` — converts `@name` → `<vp>name`
- `store.DenormalizeParams(s, prefix, params)` — `<vp>N` → `@account` (named mode)
- `store.DenormalizeVars(s, prefix)` — `<vp>N` → `@1` (positional mode)

### Resolver Scoring

`resolver.Resolve(key, links)` finds the best matching link:
- Literal path segment: **10 points**
- Variable path segment: **1 point**
- Most specific (highest score) wins
- URL passthrough: if the input contains `://`, it is returned as-is

### Group Storage

Groups store **URL templates** (not link keys). At `group create/add` time, link keys are resolved to URL templates and stored. This means:
- `resolveGroupEntries()` in `group.go` handles the link→URL conversion
- Opening a group directly resolves the stored URL templates
- Doctor checks for empty URL entries only (no link key validation needed)

### History Storage

History is stored as **append-only JSONL** (newline-delimited JSON), split by type:
- `profiles/{name}/history/link.jsonl`
- `profiles/{name}/history/group.jsonl`
- `profiles/{name}/history/url.jsonl`

Key functions in `store/history_store.go`:
- `AppendHistory(path, entry, size, dedup)` — dedup strategies: `none`, `consecutive`, `all`
- `RecentTargets(path)` — returns unique targets in MRU order (for tab completion)
- `compact` command runs erasedups + size limit across all three files

### Config Layering

`config.Load()` returns a `GlobalConfig` with profile overrides applied:
1. Load `~/.config/zebro/config.yaml` (global)
2. Apply defaults via `applyConfigDefaults`
3. Load active profile's `config.yaml`
4. Profile non-empty fields override global fields

Use `config.LoadGlobal()` when you need raw global values (no profile overlay).

### Profile System

- Active profile stored in `~/.config/zebro/.current_profile`
- `--profile/-p` global flag overrides the active profile for a single command
- Each profile has `links.yaml`, `groups.yaml`, and `history/`

## CLI Style Guide

- **Noun-first** commands: `zebro link create`, `zebro group add`, `zebro profile use`
- Subcommands without args → print help (not error):
  ```go
  if len(args) == 0 {
      return cmd.Help()
  }
  ```
- Custom help for top-level resource commands (link, group, profile) using `SetHelpFunc` with tabwriter
- `SortFlags = false` on commands where flag order matters (open, history list)
- Tab completion for all argument positions; `open -l/-g` completion uses MRU history
- `--dry-run` on `open` prints URLs instead of opening (does not record history)

## Current State (v1.1.0, branch: feat/v1.1.0)

All features implemented and committed:
- link/group CRUD with variable patterns + `rename`
- group URLs stored as URL templates (not link keys); `--url` flag for direct URLs
- open with `-l/-g/-u` flags, `open_default` config (link/group/url), `--dry-run`
- history: JSONL append-only, type-split files, MRU completion, `history_size`/`history_dedup` config
- profile CRUD with backup/restore
- config get/set with profile/global scope
- doctor for diagnostics
- shell completion (bash/zsh/fish)

**Pending for release**: merge feat/v1.1.0 → main, GitHub remote setup, Homebrew tap repo creation, GoReleaser run.
