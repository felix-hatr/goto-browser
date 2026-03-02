package cli

import (
	"fmt"
	"strings"

	"github.com/felix-hatr/goto-browser/internal/browser"
	"github.com/felix-hatr/goto-browser/internal/config"
	"github.com/felix-hatr/goto-browser/internal/resolver"
	"github.com/felix-hatr/goto-browser/internal/store"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Short:   "Run diagnostics on your zebro setup",
	Long:    "Check your zebro setup for common configuration issues.",
	Example: `  $ zebro doctor
  $ zebro doctor -p work`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hasWarning := false

		check := func(ok bool, label, detail string) {
			if ok {
				fmt.Printf("[OK]   %s\n", label)
			} else {
				fmt.Printf("[WARN] %s\n", label)
				if detail != "" {
					fmt.Printf("       -> %s\n", detail)
				}
				hasWarning = true
			}
		}

		// 1. Config file
		cfg, err := config.Load()
		check(err == nil, "config file: "+config.ConfigFile(), errStr(err))
		if err != nil {
			return nil
		}

		// 2. Active profile exists
		profileName := cfg.ActiveProfile
		if profileFlag != "" {
			profileName = profileFlag
		}
		profileExists := config.ProfileExists(profileName)
		check(profileExists, fmt.Sprintf("active profile %q exists", profileName), "run: zebro profile create "+profileName)

		if profileExists {
			// 3. Links file valid
			linksPath := config.ProfileLinksFile(profileName)
			linkList, err := store.ListLinks(linksPath)
			check(err == nil, "links.yaml syntax valid", errStr(err))

			// 4. Groups file valid
			groupsPath := config.ProfileGroupsFile(profileName)
			groups, err := store.LoadGroups(groupsPath)
			check(err == nil, "groups.yaml syntax valid", errStr(err))

			// 5b. Group link references
			if groups != nil {
				r := resolver.New(cfg.VariablePrefix)

				for posName, entry := range groups.Groups {
					displayName := store.DenormalizeParams(posName, cfg.VariablePrefix, entry.Params)
					// Skip variable groups (named or positional) — validated at open time
					if len(entry.Params) > 0 || store.HasVars(posName) {
						check(true, fmt.Sprintf("group %q: variable group (validated at open time)", displayName), "")
						continue
					}
					var missing []string
					for _, ref := range entry.Links {
						if strings.Contains(ref, "://") {
							continue // direct URL, always valid
						}
						if _, err := r.Resolve(ref, linkList); err != nil {
							missing = append(missing, store.DenormalizeVars(ref, cfg.VariablePrefix))
						}
					}
					if len(missing) > 0 {
						check(false,
							fmt.Sprintf("group %q: all links resolvable", displayName),
							fmt.Sprintf("unresolvable refs: %s", strings.Join(missing, ", ")),
						)
					} else {
						check(true, fmt.Sprintf("group %q: all links resolvable", displayName), "")
					}
				}
			}
		}

		// 7. osascript available
		asAvailable := browser.CheckAppleScript()
		check(asAvailable, "osascript (AppleScript) available", "AppleScript is required for browser control on macOS")

		// 8. Configured browser
		if asAvailable {
			browserName := cfg.Browser
			switch browserName {
			case "arc":
				arcOK := browser.CheckArcInstalled()
				check(arcOK, "Arc browser installed", "Arc not found; install from https://arc.net")
				if arcOK {
					fmt.Printf("[INFO] Arc: AppleScript tab control may be limited; open fallback is available\n")
				}
			case "chrome":
				check(true, "browser: Google Chrome (AppleScript)", "")
			case "brave":
				check(true, "browser: Brave Browser (AppleScript)", "")
			case "edge":
				check(true, "browser: Microsoft Edge (AppleScript)", "")
			case "whale":
				check(true, "browser: Naver Whale (AppleScript)", "")
			case "safari":
				check(true, "browser: Safari (AppleScript)", "")
			default:
				check(false, fmt.Sprintf("browser: %q", browserName), "unknown browser; run: zebro config set browser <chrome|brave|edge|arc|safari|whale>")
			}
		}

		if hasWarning {
			fmt.Println("\ndoctor found warnings above")
		} else {
			fmt.Println("\nall checks passed")
		}
		return nil
	},
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
