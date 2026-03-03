package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/felix-hatr/goto-browser/internal/resolver"
	"github.com/felix-hatr/goto-browser/internal/store"
	"github.com/spf13/cobra"
)

var (
	profileFlag string

	// Version is set via ldflags at build time.
	Version = "dev"
)

// Execute is the CLI entry point.
func Execute(version string) {
	Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:          "zebro",
	Short:        "goto-browser — URL shortcut manager",
	SilenceUsage: true,
}

const zebroHelpTemplate = `{{with (or .Long .Short)}}{{. | trimRightSpace}}

{{end -}}
USAGE{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} <subcommand> [flags]{{end}}{{if .HasAvailableSubCommands}}

COMMANDS{{range .Commands}}{{if .IsAvailableCommand}}
  {{colonpad .Name (add .NamePadding 3)}}{{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

FLAGS
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasAvailableInheritedFlags}}

INHERITED FLAGS
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasExample}}

EXAMPLES
{{.Example | trimRightSpace}}{{end}}{{if .HasAvailableSubCommands}}

LEARN MORE
  Use "{{.CommandPath}} <subcommand> --help" for more information about a command.{{end}}
`

func init() {
	cobra.AddTemplateFunc("add", func(a, b int) int { return a + b })
	cobra.AddTemplateFunc("colonpad", func(s string, n int) string {
		s = s + ":"
		if len(s) >= n {
			return s
		}
		return s + strings.Repeat(" ", n-len(s))
	})

	rootCmd.PersistentFlags().StringVarP(&profileFlag, "profile", "p", "", "Profile to use (overrides active profile)")
	rootCmd.RegisterFlagCompletionFunc("profile", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		profiles, err := config.ListProfiles()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return profiles, cobra.ShellCompDirectiveNoFileComp
	})
	rootCmd.Version = Version
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	rootCmd.SetHelpTemplate(zebroHelpTemplate)

	rootCmd.AddCommand(
		linkCmd,
		groupCmd,
		profileCmd,
		openCmd,
		configCmd,
		doctorCmd,
		historyCmd,
		completionCmd,
	)

	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd != rootCmd {
			defaultHelp(cmd, args)
			return
		}

		p := "@"
		if cfg, err := config.Load(); err == nil {
			p = cfg.VariablePrefix
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		fmt.Fprintln(w, "zebro — URL shortcut manager with variable patterns.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "USAGE")
		fmt.Fprintln(w, "  zebro <command> <subcommand> [flags]")
		fmt.Fprintln(w, "")

		fmt.Fprintln(w, "COMMANDS")
		fmt.Fprintln(w, "  open:\tOpen a link or group in the browser")
		fmt.Fprintln(w, "")

		fmt.Fprintln(w, "RESOURCE COMMANDS")
		fmt.Fprintln(w, "  link:\tManage links")
		fmt.Fprintln(w, "  group:\tManage groups")
		fmt.Fprintln(w, "  profile:\tManage profiles")
		fmt.Fprintln(w, "")

		fmt.Fprintln(w, "UTILITY COMMANDS")
		fmt.Fprintln(w, "  config:\tManage configuration")
		fmt.Fprintln(w, "  history:\tView and manage open history")
		fmt.Fprintln(w, "  doctor:\tRun diagnostics on your zebro setup")
		fmt.Fprintln(w, "  completion:\tGenerate shell completion script")
		fmt.Fprintln(w, "")

		fmt.Fprintln(w, "FLAGS")
		fmt.Fprintln(w, "  -h, --help\tShow help for command")
		fmt.Fprintln(w, "  -p, --profile string\tProfile to use (overrides active profile)")
		fmt.Fprintln(w, "  -v, --version\tShow zebro version")
		fmt.Fprintln(w, "")

		fmt.Fprintf(w, "EXAMPLES\n")
		fmt.Fprintf(w, "  $ zebro link create github/%[1]saccount/%[1]srepo https://github.com/%[1]saccount/%[1]srepo\n", p)
		fmt.Fprintf(w, "  $ zebro open github/octocat/hello-world\n")
		fmt.Fprintf(w, "  $ zebro open -g morning\n")
		fmt.Fprintln(w, "")

		fmt.Fprintln(w, "LEARN MORE")
		fmt.Fprintln(w, "  Use `zebro <command> --help` for more information about a command.")
		fmt.Fprintln(w, "  Tip: alias g='zebro open'")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "SHELL COMPLETION")
		switch detectShell() {
		case "fish":
			fmt.Fprintln(w, "  zebro completion -s fish > ~/.config/fish/completions/zebro.fish")
		case "bash":
			fmt.Fprintln(w, "  echo 'source <(zebro completion)' >> ~/.bashrc")
		default: // zsh
			fmt.Fprintln(w, "  echo 'source <(zebro completion)' >> ~/.zshrc")
		}

		w.Flush()
	})
}

// displayVar renders a stored key/URL for output based on the configured display mode.
// mode "named"      → DenormalizeParams (shows @account/@repo)
// mode "positional" → DenormalizeVars   (shows @1/@2)
func displayVar(s, prefix string, params []string, mode string) string {
	if mode == "positional" {
		return store.DenormalizeVars(s, prefix)
	}
	return store.DenormalizeParams(s, prefix, params)
}

// formatParams returns "@1=account, @2=repo" style string, or "" if no params.
func formatParams(prefix string, params []string) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = fmt.Sprintf("%s%d=%s", prefix, i+1, p)
	}
	return strings.Join(parts, ", ")
}

// resolveLinkURL returns the display URL for a stored link ref.
// Tries direct map lookup first (for exact keys and variable templates),
// then falls back to the resolver (for concrete values like jira/PROJ-1).
func resolveLinkURL(ref string, links []store.Link, variablePrefix string) string {
	linkMap := make(map[string]store.Link, len(links))
	for _, l := range links {
		linkMap[l.Key] = l
	}
	if link, ok := linkMap[ref]; ok {
		return store.DenormalizeVars(link.URL, variablePrefix)
	}
	r := resolver.New(variablePrefix)
	if result, err := r.Resolve(ref, links); err == nil {
		return result.URL
	}
	return ""
}

// recentSet loads MRU targets from a history file and returns them as a set.
// Returns the ordered recent slice and a set for O(1) membership tests.
func recentSet(historyPath string) ([]string, map[string]bool) {
	recent := store.RecentTargets(historyPath)
	set := make(map[string]bool, len(recent))
	for _, t := range recent {
		set[t] = true
	}
	return recent, set
}

// backupFile copies src to src+".bak". If src does not exist, it is a no-op.
func backupFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(path+".bak", data, 0600)
}

// currentProfile returns the active profile name, respecting the --profile flag.
func currentProfile() (string, *config.GlobalConfig, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", nil, err
	}
	profile := cfg.ActiveProfile
	if profileFlag != "" {
		profile = profileFlag
	}
	if !config.ProfileExists(profile) {
		return "", nil, fmt.Errorf("profile %q does not exist (run: zebro profile create %s)", profile, profile)
	}
	return profile, cfg, nil
}

