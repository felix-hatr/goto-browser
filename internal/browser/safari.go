package browser

import (
	_ "embed"
	"fmt"
	"os/exec"
	"strings"
)

//go:embed scripts/safari_new_tab.applescript
var safariNewTabScript string

//go:embed scripts/safari_new_window.applescript
var safariNewWindowScript string

// SafariBrowser handles Safari via AppleScript.
type SafariBrowser struct{}

func (b *SafariBrowser) Name() string { return "safari" }

func (b *SafariBrowser) OpenURL(url string, opts OpenOptions) error {
	return b.runScript(url, opts.NewWindow)
}

func (b *SafariBrowser) OpenURLs(urls []string, opts OpenOptions) error {
	if len(urls) == 0 {
		return nil
	}

	// First URL in new window
	if err := b.runScript(urls[0], true); err != nil {
		return err
	}

	// Remaining as tabs
	for _, u := range urls[1:] {
		if err := b.runScript(u, false); err != nil {
			return fmt.Errorf("opening %q: %w", u, err)
		}
	}
	return nil
}

func (b *SafariBrowser) runScript(url string, newWindow bool) error {
	var tmpl string
	if newWindow {
		tmpl = safariNewWindowScript
	} else {
		tmpl = safariNewTabScript
	}

	script := strings.ReplaceAll(tmpl, "PLACEHOLDER_URL", url)
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("safari applescript: %w\n%s", err, string(out))
	}
	return nil
}
