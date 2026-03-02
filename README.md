# goto-browser (zebro)

Terminal URL shortcut manager for macOS. Store URL patterns with variables and open them instantly in your browser.

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
# Add links (supports @variable patterns)
zebro link create github                     https://github.com
zebro link create github/@account/@repo     https://github.com/@account/@repo
zebro link create jira/@ticket              https://company.atlassian.net/browse/@ticket

# Open by key (variables resolved from path segments)
zebro open github/octocat/hello-world
zebro open jira/PROJ-123

# Groups — open multiple tabs at once in a new window
zebro group create morning -l github -l jira/PROJ-100
zebro open -g morning
```

## Commands

### Links
```
zebro link create <key> <url> [-d <description>]
zebro link list
zebro link view <key>
zebro link delete <key>
zebro link clear [--force]
```

### Groups
```
zebro group create <name> [-l <link-key>...] [-d <description>]
zebro group list
zebro group view <name>
zebro group add <name> -l <link-key> [-l <link-key>...] [--at <position>]
zebro group remove <name> [-l <link-key>...] [--at <position>]
zebro group delete <name>
zebro group clear [--force]
```

### Profiles
```
zebro profile create <name> [-d <description>]
zebro profile list
zebro profile view [name]
zebro profile use <name>
zebro profile delete <name>
zebro profile backup <name>
zebro profile restore <name>
```

### Open
```
zebro open <key>             # open a link (default)
zebro open -l <key>          # explicitly open a link
zebro open -g <name>         # open a group
zebro open ... -n            # open in a new window
zebro open ... -t            # open in a new tab
zebro open ... -b <browser>  # use a specific browser
zebro open ... --dry-run     # print URL(s) without opening
```

### Config
```
zebro config list
zebro config get <key>
zebro config set <key> <value>
zebro config set -g <key> <value>   # set global (not profile-specific)
```

Config keys:

| Key | Values | Default |
|-----|--------|---------|
| `browser` | chrome, brave, edge, arc, safari, whale | chrome |
| `variable_prefix` | any single non-alphanumeric char | @ |
| `variable_display` | named, positional | named |
| `open_mode` | new_tab, new_window | new_tab |
| `open_default` | link, group | link |
| `profile_delete_mode` | backup, permanent | backup |
| `profile_view_mode` | summary, detail | summary |
| `description` | any string | _(profile only)_ |

### Diagnostics
```
zebro doctor
```

## Global Flags

```
-p, --profile <name>   Use a specific profile for this command
    --dry-run          Show what would happen without opening the browser (open cmd only)
-v, --version          Show version
```

## Data Storage

All data stored in `~/.zebro/`:

```
~/.zebro/
├── config.yaml
├── .current_profile
└── profiles/
    └── default/
        ├── config.yaml
        ├── links.yaml
        └── groups.yaml
```

Files are plain YAML — easy to back up with iCloud or git.

## Shell Completion

```bash
# Zsh
echo 'source <(zebro completion)' >> ~/.zshrc && source ~/.zshrc

# Bash
echo 'source <(zebro completion)' >> ~/.bashrc

# Fish
zebro completion -s fish > ~/.config/fish/completions/zebro.fish
```

## Variable Syntax

Variables in link patterns start with `@` (configurable via `variable_prefix`):

```
Pattern:  github/@account/@repo
Input:    github/octocat/hello-world
Result:   https://github.com/octocat/hello-world
```

Variables are matched by position — `@account` and `@repo` are labels only. The name is shown in output when `variable_display` is `named`; use `positional` to show `@1/@2` style instead.

The resolver scores matches by specificity: literal path segments score 10× more than variable segments, so more specific patterns always win.

## Profiles

Profiles are isolated sets of links and groups. Useful for work vs. personal or different projects.

```bash
zebro profile create work -d "Work profile"
zebro profile use work
zebro link create jira/@ticket https://work.atlassian.net/browse/@ticket
zebro profile use default
```

## License

MIT
