package cli

import (
	"path/filepath"

	"github.com/petsagouris/specshelf/internal/vault"
	"github.com/spf13/cobra"
)

// newInitCommand creates the vault layout and prints its root.
func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the vault directory structure.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			if err := vault.Init(resolution.Root); err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte(resolution.Root + "\n"))
			return err
		},
	}
}

// newCreateCommand builds `new` (a spec) or `decision`, which differ only in the
// kind they create and the reference tool's aliases for the multi-value flags.
func newCreateCommand(use, kind, short string) *cobra.Command {
	var repos, tags, sessions stringList
	var current bool
	var status string

	cmd := &cobra.Command{
		Use:   use + " TITLE",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			ids, err := sessionValues(sessions, current)
			if err != nil {
				return err
			}
			path, err := vault.Create(resolution.Root, vault.NewDocument{
				Kind:     kind,
				Title:    args[0],
				Repos:    splitValues(repos),
				Tags:     splitValues(tags),
				Sessions: ids,
				Status:   status,
			})
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte(path + "\n"))
			return err
		},
	}

	flags := cmd.Flags()
	flags.Var(&repos, "repo", "Repo name(s), comma-separated or repeated.")
	flags.Var(&repos, "repos", "Alias of --repo.")
	flags.Var(&tags, "tag", "Tag(s), comma-separated or repeated.")
	flags.Var(&tags, "tags", "Alias of --tag.")
	flags.Var(&sessions, "codex-session", "Codex thread/session id(s), comma-separated or repeated.")
	flags.BoolVar(&current, "current-codex-session", false, "Attach CODEX_THREAD_ID from the current Codex run.")
	flags.StringVar(&status, "status", "active", "Document status.")
	return cmd
}

// newLinkSessionCommand attaches session ids to one document and prints its path.
func newLinkSessionCommand() *cobra.Command {
	var sessions stringList
	var current bool

	cmd := &cobra.Command{
		Use:   "link-session REF",
		Short: "Attach Codex session ids to one document.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			ids, err := sessionValues(sessions, current)
			if err != nil {
				return err
			}
			path, err := vault.LinkSessions(resolution.Root, args[0], ids)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte(path + "\n"))
			return err
		},
	}

	flags := cmd.Flags()
	flags.Var(&sessions, "codex-session", "Codex thread/session id(s), comma-separated or repeated.")
	flags.BoolVar(&current, "current-codex-session", false, "Attach CODEX_THREAD_ID from the current Codex run.")
	return cmd
}

// newIndexCommand regenerates index.jsonl, and with --print reports its path.
func newIndexCommand() *cobra.Command {
	var print bool
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Regenerate index.jsonl.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			if err := vault.Init(resolution.Root); err != nil {
				return err
			}
			if err := vault.WriteIndex(resolution.Root); err != nil {
				return err
			}
			if !print {
				return nil
			}
			_, err = cmd.OutOrStdout().Write([]byte(filepath.Join(resolution.Root, "index.jsonl") + "\n"))
			return err
		},
	}
	cmd.Flags().BoolVar(&print, "print", false, "Print the index path after writing.")
	return cmd
}
