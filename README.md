# goto-browser (zebro)

Terminal URL shortcut manager for macOS. Store URL patterns with variables and open them instantly in your browser.

Inspired by goat link / go link — a personal local link CLI.

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
# Add links (supports :variable patterns)
zebro add link github                 https://github.com
zebro add link github/:account/:repo  https://github.com/:account/:repo
zebro add link jira/:ticket           https://company.atlassian.net/browse/:ticket

# Open by key (variables resolved from path segments)
zebro open github/octocat/hello-world
zebro open jira/PROJ-123

# Use aliases for shorter input
zebro add alias gh github
zebro open gh/octocat/hello-world     # expands to github/octocat/hello-world

# Groups — open multiple tabs at once in a new window
zebro add group morning github jira/PROJ-100
zebro open -g morning
```

## Commands

### Links
```
zebro add link <key> <url> [--description/-d <desc>]
zebro ls link [--search <query>]
zebro get link <key>
zebro rm link <key>
```

### Aliases
```
zebro add alias <name> <link-key>
zebro ls alias
zebro get alias <name>
zebro rm alias <name>
```

### Groups
```
zebro add group <name> [link-key...] [--description/-d <desc>]
zebro ls group
zebro get group <name>
zebro rm group <name>
zebro append group <name> <link-key...>
```

### Profiles
```
zebro add profile <name> [--description/-d <desc>]
zebro ls profile
zebro get profile [name]
zebro switch profile <name>
zebro rm profile <name>
```

### Config
```
zebro config get <key>
zebro config set <key> <value>
zebro config list
```

Config keys: `browser`, `browser_profile`, `variable_prefix`, `open_mode`

Supported browsers: `chrome`, `brave`, `edge`, `arc`, `safari`, `whale`

### Open
```
zebro open <key> [--new-window/-w] [--browser/-b <browser>]
zebro open -g <name> [--new-window/-w] [--browser/-b <browser>]
```

### Diagnostics
```
zebro doctor
```

## Global Flags

```
-P, --profile <name>   Use a specific profile for this command
    --dry-run          Show what would happen without opening the browser
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
        ├── aliases.yaml
        └── groups.yaml
```

Files are plain YAML — easy to back up with iCloud or git.

## Shell Completion

```bash
# Zsh
zebro completion zsh >> ~/.zshrc && source ~/.zshrc

# Bash
zebro completion bash >> ~/.bashrc

# Fish
zebro completion fish > ~/.config/fish/completions/zebro.fish
```

## Variable Syntax

Variables in link patterns start with `:` (configurable via `variable_prefix`):

```
Pattern:  github/:account/:repo
Input:    github/octocat/hello-world
Result:   https://github.com/octocat/hello-world
```

Changing `variable_prefix` affects how variables are displayed and entered, but stored data remains compatible — variables are saved internally with a canonical token and denormalized at runtime.

The resolver scores matches by specificity: literal segments score 10× more than variable segments, so more specific patterns always win.

## License

MIT
