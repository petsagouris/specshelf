package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/petsagouris/specshelf/internal/config"
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

// commandEditor runs one editor program on a document path. It is the CLI's
// implementation of vault.Editor, so the editor handoff has one seam.
type commandEditor []string

func (e commandEditor) Edit(path string) error {
	args := make([]string, 0, len(e))
	args = append(args, e[1:]...)
	args = append(args, path)
	cmd := exec.Command(e[0], args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// editorCommand resolves the editor `edit` opens: $VISUAL, then $EDITOR, then
// the configuration file's editor setting, split on whitespace so an editor
// carrying flags ("code -w") works. "None set" is a runtime error, not a usage
// error: the command line is well formed, the environment is not.
func editorCommand() ([]string, error) {
	for _, name := range []string{"VISUAL", "EDITOR"} {
		if fields := strings.Fields(os.Getenv(name)); len(fields) > 0 {
			return fields, nil
		}
	}
	path, err := config.Path()
	if err != nil {
		return nil, err
	}
	file, _, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	if file.Editor != nil {
		if fields := strings.Fields(*file.Editor); len(fields) > 0 {
			return fields, nil
		}
	}
	return nil, errors.New("no editor: set VISUAL or EDITOR, or the editor setting in the config file")
}

// newEditCommand opens one document in the user's editor and prints its path.
func newEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit REF",
		Short: "Open one document in your editor.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			editor, err := editorCommand()
			if err != nil {
				return err
			}
			path, err := vault.Edit(resolution.Root, args[0], commandEditor(editor))
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte(path + "\n"))
			return err
		},
	}
}

// newRemoveCommand removes one document, or reports it with --dry-run.
func newRemoveCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "rm REF",
		Short: "Remove one document.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			var path string
			if dryRun {
				found, err := vault.Find(resolution.Root, args[0])
				if err != nil {
					return err
				}
				path = found.Path
			} else {
				path, err = vault.Remove(resolution.Root, args[0])
				if err != nil {
					return err
				}
			}
			_, err = cmd.OutOrStdout().Write([]byte(path + "\n"))
			return err
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Report the document that would be removed without removing it.")
	return cmd
}

// newStatusCommand sets one document's status and stamps updated to today.
func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status REF STATUS",
		Short: "Set one document's status.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			path, err := vault.SetStatus(resolution.Root, args[0], args[1])
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write([]byte(path + "\n"))
			return err
		},
	}
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
