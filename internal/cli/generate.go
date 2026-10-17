package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// newCompletionCommand generates a shell completion script for one of the four
// shells cobra supports, straight from the command tree (design D4).
func newCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "completion SHELL",
		Short:     "Generate a shell completion script.",
		Long:      "Generate a completion script for bash, zsh, fish or PowerShell from the command definitions.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(out)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(out)
			default:
				return usagef("invalid argument %q for \"completion\": must be bash, zsh, fish, or powershell", args[0])
			}
		},
	}
}

// newManCommand generates a roff man page for the root command and every
// subcommand into DIR, from the command definitions themselves (design D4).
func newManCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "man [DIR]",
		Short: "Generate roff man pages.",
		Long:  "Generate a roff man page for the root command and every subcommand into DIR (the working directory by default).",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			// A nil Title lets cobra derive each page's title from its command
			// path, so a subcommand page is named SPECSHELF-LIST rather than
			// repeating the root's.
			header := &doc.GenManHeader{
				Section: "1",
				Source:  "specshelf",
				Manual:  "specshelf Manual",
			}
			return doc.GenManTree(cmd.Root(), header, dir)
		},
	}
}
