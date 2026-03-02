package browser

// Browser defines the interface for opening URLs in a browser.
type Browser interface {
	// OpenURL opens a single URL.
	OpenURL(url string, opts OpenOptions) error
	// OpenURLs opens multiple URLs, typically as tabs in a window.
	OpenURLs(urls []string, opts OpenOptions) error
	// Name returns the canonical browser name.
	Name() string
}

// OpenOptions controls how URLs are opened.
type OpenOptions struct {
	NewWindow bool
}
