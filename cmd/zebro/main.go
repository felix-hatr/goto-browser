package main

import (
	"github.com/felix-hatr/goto-browser/internal/cli"
)

// Version is set via -ldflags at build time.
var Version = "dev"

func main() {
	cli.Execute(Version)
}
