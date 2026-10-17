// Package cli turns command-line arguments into vault operations and writes the
// results to stdout and stderr.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// Main runs the command line with the process arguments and streams, and
// returns the process exit code.
func Main() int {
	return Run(os.Args[1:], os.Stdout, os.Stderr)
}

// Run executes a single command invocation against explicit streams.
func Run(args []string, stdout, stderr io.Writer) int {
	root := NewRootCommand()
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	if err := root.Execute(); err != nil {
		_, _ = fmt.Fprintf(stderr, "specshelf: %s\n", err)
		return 1
	}
	return 0
}

// NewRootCommand builds the command tree.
func NewRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "specshelf",
		Short:        "Manage a global Markdown vault of specs and decisions for AI coding agents.",
		Version:      Version(),
		SilenceUsage: true,
	}
}
