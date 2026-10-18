package cli

import (
	"github.com/petsagouris/specshelf/internal/tui"
	"github.com/spf13/cobra"
)

// newUICommand starts the interactive session over the resolved vault. The
// session is handed the vault store — internal/vault is the only package that
// touches vault files — and the same editor implementation `edit` uses.
// The terminal check comes before resolving an editor: a redirected run is not
// interactive whatever editor is configured, so it reports the non-terminal
// error rather than a missing editor. tui.Run makes the same check; here it
// runs early enough that the editor is never resolved for a run that cannot
// start.
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
			if !isTerminal(cmd.OutOrStdout()) {
				return tui.NotTerminalError{}
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
