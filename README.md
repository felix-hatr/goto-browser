# zebro

Terminal URL shortcut manager for macOS.
Link keys to URLs, resolve variables from the path, and open them instantly in your browser.

```bash
zebro link create github/@account/@repo  https://github.com/@account/@repo
zebro open github/octocat/hello-world    # opens https://github.com/octocat/hello-world
```

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
# 1. Create links
zebro link create github                     https://github.com
zebro link create github/@account/@repo     https://github.com/@account/@repo
zebro link create jira/@ticket              https://company.atlassian.net/browse/@ticket

# 2. Open them
zebro open github                    # https://github.com
zebro open github/octocat            # https://github.com/octocat  (wait — no @repo, won't match the 2-var pattern)
zebro open github/octocat/hello      # https://github.com/octocat/hello
zebro open jira/PROJ-123             # https://company.atlassian.net/browse/PROJ-123

# 3. Group links and open them all at once
zebro group create morning -l github -l jira/PROJ-100
zebro open -g morning                # opens both in a new window
```

---

## Commands

### `zebro open`

```
zebro open <key>              open a link (default)
zebro open -l <key>           explicitly open a link
zebro open -g <name>          open a group
zebro open ... -n             open in a new window
zebro open ... -t             open in a new tab
zebro open ... -b <browser>   use a specific browser
zebro open ... --dry-run      print URL(s) without opening
```

### `zebro link`

```
zebro link create <key> <url> [-d <description>]    add or update a link
zebro link list                                      list all links
zebro link view <key>                                show link details
zebro link delete <key>                              remove a link
zebro link clear                                     remove all links
```

### `zebro group`

```
zebro group create <name> [-l <link-key>...] [-d <description>]    create a group
zebro group list                                                    list all groups
zebro group view <name>                                             show group details
zebro group add <name> -l <link-key> [...] [--at <position>]       add links to a group
zebro group remove <name> [-l <link-key>...] [--at <position>]     remove links from a group
zebro group delete <name>                                           remove a group
zebro group clear                                                   remove all groups
```

### `zebro profile`

Profiles are isolated sets of links and groups — useful for work vs. personal, or different projects.

```
zebro profile list                             list all profiles
zebro profile view [name] [-d] [-s]           show profile details (active profile if no name given)
zebro profile create <name> [-d <desc>] [-s <source>]   create a new profile
zebro profile use <name>                       switch the active profile
zebro profile rename <old> <new>               rename a profile
zebro profile delete <name>                    delete a profile (follows profile_delete_mode config)
zebro profile delete <name> --force            delete immediately, no backup
zebro profile delete <name> --backup           always back up before deleting
zebro profile delete <name> --purge            delete profile and all its backups permanently
```

**Profile backups:**

```
zebro profile backup list [name]              list all backups
zebro profile backup view <name> <id>         show contents of a backup
zebro profile backup create <name>            take a manual snapshot
zebro profile backup restore <name> [id]      restore from backup (latest if no id given)
zebro profile backup delete <name> <id>       delete a specific backup
zebro profile backup clear <name>             delete all backups for a profile
```

### `zebro config`

```
zebro config list                    show all settings (current profile)
zebro config get <key>               get a single value
zebro config set <key> <value>       set a value for the current profile
zebro config set -g <key> <value>    set a global value (all profiles)
```

| Key | Values | Default | Note |
|-----|--------|---------|------|
| `browser` | chrome, brave, edge, arc, safari, whale | chrome | |
| `variable_prefix` | any single symbol (e.g. `@`, `^`, `:`) | `@` | |
| `variable_display` | named, positional | named | affects output only |
| `open_mode` | new_tab, new_window | new_tab | |
| `open_default` | link, group | link | what `zebro open <arg>` resolves to |
| `profile_delete_mode` | backup, permanent | backup | |
| `profile_view_mode` | summary, detail | summary | |
| `description` | any string | — | profile only |

### `zebro doctor`

Runs diagnostics — checks for broken group references, missing links, and config issues.

```
zebro doctor
```

---

## Variables

Variables in link patterns start with `@` (the default `variable_prefix`):

```
Pattern:  github/@account/@repo
Input:    github/octocat/hello-world
Result:   https://github.com/octocat/hello-world
```

Matching is positional — `@account` and `@repo` are just labels. The same pattern stored as `github/@1/@2` works identically.

The resolver picks the most specific match: literal path segments score 10× more than variable segments, so `github/octocat` always beats `github/@account` when both could match.

Variable groups work the same way — variables in the group name map to the link keys:

```bash
zebro group create dev/@account -l github/@account -l jira/@ticket
zebro open -g dev/myorg              # @account=myorg for all links in the group
```

---

## Global Flags

```
-p, --profile <name>    use a specific profile for this command (overrides active profile)
-v, --version           show version
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

---

## Data Storage

All data is stored in `~/.config/zebro/` (follows XDG Base Directory spec; override with `$XDG_CONFIG_HOME`):

```
~/.config/zebro/
├── config.yaml            global config
├── .current_profile       active profile name
├── profiles/
│   ├── default/
│   │   ├── config.yaml    profile-level config overrides
│   │   ├── links.yaml
│   │   └── groups.yaml
│   └── work/
│       ├── config.yaml
│       ├── links.yaml
│       └── groups.yaml
└── profiles/.bak/         profile backups (timestamped snapshots)
    └── default.20240101-120000/
        ├── links.yaml
        └── groups.yaml
```

Files are plain YAML — easy to back up with iCloud Drive or git.

---

## License

MIT — see [LICENSE](LICENSE)
