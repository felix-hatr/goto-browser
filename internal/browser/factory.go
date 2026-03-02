package browser

import (
	"fmt"
	"strings"
)

// New returns a Browser driver for the given browser name.
// Supported names: chrome, brave, edge, arc, safari, whale
func New(name string) (Browser, error) {
	switch strings.ToLower(name) {
	case "chrome":
		return newChromeBrowser("chrome", "Google Chrome"), nil
	case "brave":
		return newChromeBrowser("brave", "Brave Browser"), nil
	case "edge":
		return newChromeBrowser("edge", "Microsoft Edge"), nil
	case "whale":
		return newChromeBrowser("whale", "Whale"), nil
	case "arc":
		return &ArcBrowser{}, nil
	case "safari":
		return &SafariBrowser{}, nil
	default:
		return nil, fmt.Errorf("unsupported browser: %q (supported: chrome, brave, edge, arc, safari, whale)", name)
	}
}
