package browser

import (
	_ "embed"
	"fmt"
	"os/exec"
	"strings"
)

//go:embed scripts/arc_new_tab.applescript
var arcNewTabScript string

//go:embed scripts/arc_new_window.applescript
var arcNewWindowScript string

// ArcBrowser handles Arc browser — tries AppleScript, falls back to open -a Arc.
type ArcBrowser struct{}

func (b *ArcBrowser) Name() string { return "arc" }

func (b *ArcBrowser) OpenURL(url string, opts OpenOptions) error {
	if err := b.tryAppleScript(url, opts.NewWindow); err != nil {
		return b.openFallback(url)
	}
	return nil
}

func (b *ArcBrowser) OpenURLs(urls []string, opts OpenOptions) error {
	if len(urls) == 0 {
		return nil
	}

	// Try AppleScript first for the first URL in a new window
	if err := b.tryAppleScript(urls[0], true); err != nil {
		// Fallback: open each URL individually
		fmt.Printf("warning: Arc AppleScript tab control unavailable, opening individually\n")
		for _, u := range urls {
			if err := b.openFallback(u); err != nil {
				return err
			}
		}
		return nil
	}

	// Open remaining as tabs
	for _, u := range urls[1:] {
		if err := b.tryAppleScript(u, false); err != nil {
			if err2 := b.openFallback(u); err2 != nil {
				return err2
			}
		}
	}
	return nil
}

func (b *ArcBrowser) tryAppleScript(url string, newWindow bool) error {
	var tmpl string
	if newWindow {
		tmpl = arcNewWindowScript
	} else {
		tmpl = arcNewTabScript
	}

	script := strings.ReplaceAll(tmpl, "PLACEHOLDER_URL", url)
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("arc applescript: %w: %s", err, string(out))
	}
	return nil
}

func (b *ArcBrowser) openFallback(url string) error {
	cmd := exec.Command("open", "-a", "Arc", url)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("open -a Arc: %w: %s", err, string(out))
	}
	return nil
}

// CheckArcInstalled checks whether Arc is installed.
func CheckArcInstalled() bool {
	cmd := exec.Command("osascript", "-e", `tell application "Finder" to return exists application "Arc"`)
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}
