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

Always use `-buildvcs=false` — avoids VCS metadata embedding issues in this environment.

## Project Structure

```
cmd/zebro/         — entry point (sets version via ldflags)
internal/
  cli/             — cobra commands (one file per resource)
    root.go        — rootCmd, global flags, isTTY/highlightKeyword helpers
    link.go        — zebro link {create,list,view,rename,delete,clear,search,export,import}
    group.go       — zebro group {create,list,view,add,remove,rename,delete,clear,search,export,import}
    profile.go     — zebro profile {create,list,view,use,rename,delete,backup,export,import}
    open.go        — zebro open (multi-arg, --no-history)
    history.go     — zebro history {list,stats,compact,clear,search}
    search.go      — zebro search (unified search)
    config.go      — zebro config {list,get,set}
    doctor.go      — zebro doctor
    backup.go      — backup helpers
    completion.go  — zebro completion
  config/          — config load/save, paths, profile I/O; validateConfigValue shared helper
  store/           — YAML read/write for links/groups; JSONL for history; variable token logic
    export_file.go — ExportFile type for import/export
  resolver/        — URL resolution with scoring; ResolveGroupEntries method
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
- `r.ResolveGroupEntries()` in `resolver/resolver.go` handles the link→URL conversion (moved from group.go in v1.2.0)
- Opening a group directly resolves the stored URL templates
- Doctor checks for empty URL entries only (no link key validation needed)

### ExportFile Format

Import/export uses a shared YAML format (`store.ExportFile`):
```yaml
version: "1"
links:        # map[key]LinkEntry (omitempty)
groups:       # map[name]GroupEntry (omitempty)
config:       # map[string]string (omitempty)
```
- Variable tokens are stored in internal `<vp>N` form (prefix-independent)
- `profile export` includes all three sections; `link/group export` includes only their section
- Import default: merge (skip conflicts); `--replace` overwrites all existing data

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
- `--no-history` on `open` opens normally but suppresses history recording
- Search commands: write to `bytes.Buffer` → tabwriter flush → `highlightKeyword()` → print
  (so ANSI codes don't corrupt column alignment)

## Current State (v1.2.0, branch: feat/v1.2.0)

All features implemented and committed:
- link/group CRUD with variable patterns + `rename`
- group URLs stored as URL templates (not link keys); `--url` flag for direct URLs
- open with repeatable `-l/-g/-u` flags (ordered), positional multi-args, `--dry-run`, `--no-history`
- `link search`, `group search`, `history search`, `zebro search` (unified) — case-insensitive, keyword highlight
- `link export/import`, `group export/import`, `profile export/import` with merge/replace modes
- history: JSONL append-only, type-split files, MRU completion, `history_size`/`history_dedup` config
- `history list` supports combining `-l/-g/-u` flags (no longer mutually exclusive)
- profile CRUD with backup/restore
- config get/set with profile/global scope; `validateConfigValue` shared helper
- `resolver.ResolveGroupEntries` (moved from cli/group.go)
- doctor for diagnostics
- shell completion (bash/zsh/fish)

