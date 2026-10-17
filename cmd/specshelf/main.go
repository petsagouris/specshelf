// Command specshelf manages a global Markdown vault of specs, cross-repo
// contracts and decisions for AI coding agents.
package main

import (
	"os"

	"github.com/petsagouris/specshelf/internal/cli"
)

func main() {
	os.Exit(cli.Main())
}
