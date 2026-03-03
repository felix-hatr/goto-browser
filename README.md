# zebro — Terminal URL Shortcut Manager for macOS

**zebro** is a fast, keyboard-driven URL shortcut manager for the macOS terminal.
Map short key paths to URL patterns, fill variables from the path, and open them instantly in any browser.

```bash
zebro link create github/@user/@repo  https://github.com/@user/@repo
zebro open github/octocat/hello-world   # → https://github.com/octocat/hello-world
```

> **Tip:** `alias g='zebro open'` — then `g jira/PROJ-123` opens your ticket in one keystroke.

## Features

- **Variable URL patterns** — `github/@user/@repo` matches `github/octocat/hello-world` positionally
- **Groups** — open multiple URLs at once in a new browser window
- **Multi-open** — `-l/-g/-u` flags are repeatable; open links, groups, and URLs in one command
- **Search** — `zebro search`, `zebro link search`, `zebro group search`, `zebro history search`
- **Import / Export** — portable YAML snapshots for links, groups, and full profiles
- **Profiles** — isolated link sets for work, personal, or per-project
- **History** — records every open; tab completion surfaces recently used items first
- **Smart resolver** — most specific match wins (literal segments beat variable segments)
- **Shell completion** — bash, zsh, fish with MRU-ordered suggestions

## Install

```bash
brew install felix-hatr/goto-browser/goto-browser
```

Or build from source:

```bash
git clone https://github.com/felix-hatr/goto-browser
cd goto-browser
make install
```

## Quick Start

```bash
# Create link shortcuts
zebro link create github                    https://github.com
zebro link create github/@user/@repo       https://github.com/@user/@repo
zebro link create jira/@ticket             https://company.atlassian.net/browse/@ticket

# Open them — variables filled from the path
zebro open github                    # https://github.com
zebro open github/octocat/hello      # https://github.com/octocat/hello
zebro open jira/PROJ-123             # https://company.atlassian.net/browse/PROJ-123
zebro open -u https://example.com    # open any URL directly

# Group links for one-command multi-tab workflows
zebro group create morning --url https://github.com --url https://notion.so
zebro open -g morning                # opens all URLs in a new window
```

---

## Commands

### `zebro open`

```
zebro open <key>                        open a link (default)
zebro open -l <key>                     explicitly open a link
zebro open -g <name>                    open a group
zebro open -u <url>                     open a direct URL
zebro open -l github -g morning         open multiple items in order (flags are repeatable)
zebro open -l github -l jira/PROJ-100  open multiple links
zebro open ... -n                       open in a new window
zebro open ... -t                       open in a new tab
zebro open ... -b <browser>             use a specific browser
zebro open ... --dry-run                print URL(s) without opening
zebro open ... --no-history             open without recording to history
```

Flags `-l`, `-g`, and `-u` are **repeatable** and processed in command-line order.
Positional arguments follow any flags and are treated per `open_default` config.
Tab completion for `-l` and `-g` shows recently opened items first (MRU order).

### `zebro link`

```
zebro link create <key> <url> [-d <description>]    add or update a link
zebro link list                                      list all links
zebro link view <key>                                show link details
zebro link search <keyword>                          search links by key, URL, or description
zebro link rename <old-key> <new-key>                rename a link key
zebro link delete <key>                              remove a link
zebro link clear                                     remove all links
zebro link export [-o <file>]                        export links to YAML (stdout if no -o)
zebro link import <file> [--replace]                 import links from YAML
```

### `zebro group`

```
zebro group create <name> [-l <key>...] [-u <url>...] [-d <desc>]    create a group
zebro group list                                                       list all groups
zebro group view <name>                                                show group details
zebro group search <keyword>                                           search groups by name or description
zebro group add <name> [-l <key>...] [-u <url>...] [--at <pos>]       add to a group
zebro group remove <name> [-l <key>...] [--at <pos>]                  remove from a group
zebro group rename <old-name> <new-name>                               rename a group
zebro group delete <name>                                              remove a group
zebro group clear                                                      remove all groups
zebro group export [-o <file>]                                         export groups to YAML
zebro group import <file> [--replace]                                  import groups from YAML
```

Use `-l` to reference a registered link key, or `-u` to add a direct URL. Both support group-level variable substitution.

### `zebro history`

```
zebro history list [-l] [-g] [-u] [-n <count>]    list history (most recent first; flags combinable)
zebro history search <keyword> [-l] [-g] [-u]     search history by target or URL
zebro history stats                               open frequency and top targets
zebro history clear                               clear all history
zebro history compact                             deduplicate and apply size limit
```

History is recorded for every successful `zebro open` (skipped on `--dry-run` or `--no-history`). Tab completion uses it to surface recently used items first. `-l/-g/-u` flags can be combined to filter multiple types.

### `zebro profile`

Profiles are isolated sets of links and groups — useful for work vs. personal, or per-project setups.

```
zebro profile create <name> [-d <desc>]       create a profile
zebro profile list                             list all profiles
zebro profile view [name]                      show profile details
zebro profile use <name>                       switch active profile
zebro profile rename <old> <new>               rename a profile
zebro profile delete <name>                    delete (follows profile_delete_mode config)
zebro profile export [name] [-o <file>]        export links + groups + config to YAML
zebro profile import <file> [--as <name>] [--force]   import from YAML
```

**Backups:**

```
zebro profile backup create <name>            snapshot current state
zebro profile backup list [name]              list snapshots
zebro profile backup restore <name> [id]      restore (latest if no id given)
zebro profile backup view <name> <id>         inspect a snapshot
zebro profile backup delete <name> <id>       delete a snapshot
zebro profile backup clear <name>             delete all snapshots
```

### `zebro search`

Search across all resources at once:

```
zebro search <keyword>    search links, groups, and history in one command
```

Results are grouped into **LINKS / GROUPS / HISTORY** sections (empty sections omitted).
Keywords are highlighted in terminal output. Also available per resource:

```
zebro link search <keyword>
zebro group search <keyword>
zebro history search <keyword>
```

### `zebro config`

```
zebro config list                    show all settings (current profile)
zebro config get <key>               get a value
zebro config set <key> <value>       set a value for the current profile
zebro config set -g <key> <value>    set a global value (all profiles)
```

| Key | Values | Default | Note |
|-----|--------|---------|------|
| `browser` | chrome, brave, edge, arc, safari, whale | chrome | |
| `variable_prefix` | any single symbol | `@` | e.g. `^`, `:`, `~` |
| `variable_display` | named, positional | named | display only |
| `open_mode` | new_tab, new_window | new_tab | |
| `open_default` | link, group, url | link | what bare `zebro open <arg>` resolves to |
| `profile_delete_mode` | backup, permanent | backup | |
| `profile_view_mode` | summary, detail | summary | |
| `history_size` | positive integer, -1 | 10000 | entries per type; -1 = unlimited |
| `history_dedup` | none, consecutive, all | none | `consecutive`: skip same-as-last; `all`: move to end |
| `description` | any string | — | profile only |

### `zebro doctor`

Checks for configuration issues and empty group entries.

```bash
zebro doctor
```

---

## How Variables Work

Variables in link patterns start with `@` (configurable via `variable_prefix`):

```
Pattern:  github/@user/@repo
Input:    github/octocat/hello-world
          └── pos 1: @user=octocat   pos 2: @repo=hello-world
Result:   https://github.com/octocat/hello-world
```

Matching is **positional** — `@user` and `@repo` are labels only. `github/@1/@2` is identical.

The resolver picks the **most specific** match: literal segments score 10× more than variable segments, so `github/octocat` always beats `github/@user` when both could match.

Variable groups work the same way:

```bash
zebro group create dev/@user -l github/@user -u https://ci.example.com/@user
zebro open -g dev/myorg    # @user=myorg substituted in all URLs
```

---

## Global Flags

```
-p, --profile <name>    use a specific profile for this command
-v, --version           print version
```

---

## Shell Completion

```bash
# Zsh
echo 'source <(zebro completion)' >> ~/.zshrc && source ~/.zshrc

# Bash
echo 'source <(zebro completion)' >> ~/.bashrc

# Fish
zebro completion -s fish > ~/.config/fish/completions/zebro.fish
```

Completions for `zebro open -l` and `zebro open -g` rank recently used items first.

---

## Data Storage

All data lives in `~/.config/zebro/` (XDG Base Directory; override with `$XDG_CONFIG_HOME`):

```
~/.config/zebro/
├── config.yaml                  global settings
├── .current_profile             active profile name
├── profiles/
│   └── default/
│       ├── config.yaml          profile-level overrides
│       ├── links.yaml           link shortcuts
│       ├── groups.yaml          URL groups
│       └── history/
│           ├── link.jsonl       link open history
│           ├── group.jsonl      group open history
│           └── url.jsonl        direct URL history
└── profiles/.bak/               timestamped profile snapshots
```

Links and groups are plain YAML — easy to version-control or sync with iCloud Drive.
History files are append-only JSONL for fast writes.

---

## License

MIT — see [LICENSE](LICENSE)
