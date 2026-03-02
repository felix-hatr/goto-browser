package cli

import (
	"fmt"
	"os"
	"strings"

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

func init() {
	rootCmd.PersistentFlags().StringVarP(&profileFlag, "profile", "p", "", "Profile to use (overrides active profile)")
	rootCmd.Version = Version

	rootCmd.AddCommand(
		linkCmd,
		aliasCmd,
		groupCmd,
		profileCmd,
		openCmd,
		configCmd,
		doctorCmd,
	)

	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == rootCmd {
			p := "@"
			if cfg, err := config.Load(); err == nil {
				p = cfg.VariablePrefix
			}
			cmd.Long = fmt.Sprintf(`zebro stores URL patterns with variables and opens them in your browser.

Examples:
  zebro link create github/%[1]saccount/%[1]srepo https://github.com/%[1]saccount/%[1]srepo
  zebro open github/octocat/hello-world

Tip: add a shell alias for quick access:
  alias g='zebro open'`, p)
		}
		defaultHelp(cmd, args)
	})
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
func resolveLinkURL(ref string, links []store.Link, aliases map[string]string, variablePrefix string) string {
	linkMap := make(map[string]store.Link, len(links))
	for _, l := range links {
		linkMap[l.Key] = l
	}
	if link, ok := linkMap[ref]; ok {
		return store.DenormalizeVars(link.URL, variablePrefix)
	}
	r := resolver.New(variablePrefix)
	if result, err := r.Resolve(ref, links, aliases); err == nil {
		return result.URL
	}
	return ""
}

// loadLinksAndAliases loads links and the alias map for the given profile.
func loadLinksAndAliases(profile string) ([]store.Link, map[string]string, error) {
	links, err := store.ListLinks(config.ProfileLinksFile(profile))
	if err != nil {
		return nil, nil, err
	}
	af, err := store.LoadAliases(config.ProfileAliasesFile(profile))
	if err != nil {
		return nil, nil, err
	}
	aliases := map[string]string{}
	if af != nil {
		aliases = af.Aliases
	}
	return links, aliases, nil
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

