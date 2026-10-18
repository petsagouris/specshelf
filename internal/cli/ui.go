package cli

import (
	"github.com/petsagouris/specshelf/internal/tui"
	"github.com/spf13/cobra"
)

// newUICommand starts the interactive session over the resolved vault. The
// session is handed the vault store — internal/vault is the only package that
// touches vault files — and the same editor implementation `edit` uses.
// tui.Run refuses to start when stdout is not a terminal, returning the error
// this command reports on stderr with the runtime exit code; nothing is written
// to the vault in that case, and the error points at the non-interactive
// commands.
func newUICommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ui",
		Short: "Browse and edit the vault in an interactive session.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			editor, err := editorCommand()
			if err != nil {
				return err
			}
			store := tui.VaultStore{Root: resolution.Root, Editor: commandEditor(editor)}
			return tui.Run(store, tui.WithInput(cmd.InOrStdin()), tui.WithOutput(cmd.OutOrStdout()))
		},
	}
}
