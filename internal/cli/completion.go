package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)


var completionShellFlag string

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for zebro.

The script enables tab completion for all zebro commands, subcommands, flags,
and dynamic values (link keys, group names, profile names, config keys).`,
	Example: `  # shell is auto-detected from $SHELL
  $ zebro completion

  # Add to your shell profile:
  #   zsh:  echo 'source <(zebro completion)' >> ~/.zshrc
  #   bash: echo 'source <(zebro completion)' >> ~/.bashrc
  #   fish: zebro completion -s fish > ~/.config/fish/completions/zebro.fish`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := completionShellFlag
		if shell == "" && len(args) > 0 {
			shell = args[0]
		}
		if shell == "" {
			shell = detectShell()
		}
		switch shell {
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		default:
			return fmt.Errorf("unsupported shell %q; supported: bash, zsh, fish\ndetected shell: %s", shell, detectShell())
		}
	},
}

func init() {
	completionCmd.Flags().StringVarP(&completionShellFlag, "shell", "s", "", "Shell type: bash, zsh, fish (default: auto-detect from $SHELL)")
	completionCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return []string{"bash", "zsh", "fish"}, cobra.ShellCompDirectiveNoFileComp
	}
}

// detectShell returns the shell name from $SHELL, defaulting to "bash".
func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "bash"
	}
	return strings.ToLower(filepath.Base(shell))
}
