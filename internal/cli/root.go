// Package cli turns command-line arguments into vault operations and writes the
// results to stdout and stderr.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/petsagouris/specshelf/internal/vault"
	"github.com/spf13/cobra"
)

// The exit codes the CLI contract defines: success, a runtime failure, and a
// usage failure such as an unknown flag or two conflicting vault selectors.
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

// usageError marks an error as a usage failure, which exits 2. Flag parse errors
// are wrapped in one; a command can wrap its own invalid arguments the same way.
type usageError struct{ err error }

func (e *usageError) Error() string { return e.err.Error() }
func (e *usageError) Unwrap() error { return e.err }

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
		return exitCode(err)
	}
	return exitOK
}

// exitCode classifies a failed command: a usage failure exits 2, anything else
// exits 1. Cobra's own flag, argument and unknown-command errors carry no type,
// so their messages are recognised here; the flag error function covers the flag
// case directly.
func exitCode(err error) int {
	var usage *usageError
	var conflict *vault.ConflictError
	if errors.As(err, &usage) || errors.As(err, &conflict) {
		return exitUsage
	}
	message := err.Error()
	for _, prefix := range []string{"unknown command", "unknown flag", "unknown shorthand flag", "required flag", "accepts ", "invalid argument"} {
		if strings.HasPrefix(message, prefix) {
			return exitUsage
		}
	}
	return exitError
}

// NewRootCommand builds the command tree.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "specshelf",
		Short:         "Manage a global Markdown vault of specs and decisions for AI coding agents.",
		Version:       Version(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &usageError{err: err}
	})
	// The reference tool has no completion command; task 4.4 adds one that
	// generates the scripts explicitly, so cobra's implicit one is off.
	root.CompletionOptions.DisableDefaultCmd = true

	flags := root.PersistentFlags()
	flags.String("home", "", "Vault directory. Defaults to SPECSHELF_HOME, the config file, or ~/.agents/spec-vault.")
	flags.String("vault", "", "A named vault declared in the config file.")
	flags.Bool("project", false, "Use the project vault found by walking up for a .specshelf directory.")

	root.AddCommand(
		newInitCommand(),
		newCreateCommand("new", "spec", "Create a new spec."),
		newCreateCommand("decision", "decision", "Create a new decision note."),
		newLinkSessionCommand(),
		newListCommand(),
		newSearchCommand(),
		newShowCommand(),
		newIndexCommand(),
		newPrimeCommand(),
		newPathCommand(),
	)
	return root
}
