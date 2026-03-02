package browser

import (
	_ "embed"
	"fmt"
	"os/exec"
	"strings"
)

//go:embed scripts/chrome_new_tab.applescript
var chromeNewTabScript string

//go:embed scripts/chrome_new_window.applescript
var chromeNewWindowScript string

// ChromeBrowser handles Chrome, Brave, and Edge via AppleScript.
type ChromeBrowser struct {
	appName string // "Google Chrome", "Brave Browser", "Microsoft Edge"
	name    string // canonical name: "chrome", "brave", "edge"
}

func newChromeBrowser(name, appName string) *ChromeBrowser {
	return &ChromeBrowser{appName: appName, name: name}
}

func (b *ChromeBrowser) Name() string { return b.name }

func (b *ChromeBrowser) OpenURL(url string, opts OpenOptions) error {
	return b.runScript(url, opts.NewWindow)
}

func (b *ChromeBrowser) OpenURLs(urls []string, opts OpenOptions) error {
	if len(urls) == 0 {
		return nil
	}

	// Open first URL in new window always (for group open)
	if err := b.runScript(urls[0], true); err != nil {
		return err
	}

	// Open remaining URLs as tabs
	for _, u := range urls[1:] {
		if err := b.runScript(u, false); err != nil {
			return fmt.Errorf("opening %q: %w", u, err)
		}
	}
	return nil
}

func (b *ChromeBrowser) runScript(url string, newWindow bool) error {
	var tmpl string
	if newWindow {
		tmpl = chromeNewWindowScript
	} else {
		tmpl = chromeNewTabScript
	}

	script := strings.ReplaceAll(tmpl, "BROWSER_APP", b.appName)
	script = strings.ReplaceAll(script, "PLACEHOLDER_URL", url)

	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("applescript error: %w\n%s", err, string(out))
	}
	return nil
}

// CheckAppleScript verifies that osascript is available.
func CheckAppleScript() bool {
	cmd := exec.Command("osascript", "-e", `return "ok"`)
	return cmd.Run() == nil
}
