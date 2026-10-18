package cli

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/petsagouris/specshelf/internal/config"
	"github.com/petsagouris/specshelf/internal/gitx"
	"github.com/spf13/cobra"
)

// newSyncCommand commits the vault and, when push is enabled, pushes it with the
// user's own git installation (design D9). The remote and the push default come
// from the configuration's [git] section; --push and --no-push override the
// default and are mutually exclusive. The report of what happened is the
// payload, so it goes to stdout; a failure is returned and printed on stderr by
// Run, and nothing is force-pushed or rewritten.
func newSyncCommand() *cobra.Command {
	var push, noPush bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Commit the vault, and push it with git when enabled.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			if push && noPush {
				return usagef("--push and --no-push are mutually exclusive")
			}

			// resolveVault has already loaded the configuration and printed its
			// warnings, so this load only reads the [git] section.
			path, err := config.Path()
			if err != nil {
				return err
			}
			file, _, err := config.Load(path)
			if err != nil {
				return err
			}
			remote, doPush := gitSettings(file)
			switch {
			case push:
				doPush = true
			case noPush:
				doPush = false
			}
			// An unconfigured remote is nothing to push, not an error: the
			// git-sync spec has the first sync in a plain directory and every
			// repeat exit 0, and only a remote that rejects a push fails. gitx
			// pushes to "origin" when no remote is named, so the check is
			// against the remote that would actually be pushed to.
			if doPush && !hasRemote(resolution.Root, remoteOrOrigin(remote)) {
				doPush = false
			}

			result, err := gitx.Sync(gitx.Options{Root: resolution.Root, Push: doPush, Remote: remote})
			if err != nil {
				return err
			}
			return writeSyncReport(cmd.OutOrStdout(), result, remote)
		},
	}

	cmd.Flags().BoolVar(&push, "push", false, "Push after committing, overriding the configured default.")
	cmd.Flags().BoolVar(&noPush, "no-push", false, "Do not push, overriding the configured default.")
	return cmd
}

// remoteOrOrigin is the remote a push would use: the configured one, or the
// "origin" gitx falls back to.
func remoteOrOrigin(remote string) string {
	if remote == "" {
		return "origin"
	}
	return remote
}

// hasRemote reports whether the vault's repository has a remote by that name. A
// vault that is not a repository yet, or that has no such remote, has nothing
// for a push to reach, so sync commits rather than failing.
func hasRemote(root, name string) bool {
	out, err := exec.Command("git", "-C", root, "remote").Output()
	if err != nil {
		return false
	}
	for _, remote := range strings.Fields(string(out)) {
		if remote == name {
			return true
		}
	}
	return false
}

// gitSettings reads the configuration's [git] section: the remote to push to and
// whether to push at all. An absent section pushes, as the documented default.
func gitSettings(file *config.File) (remote string, push bool) {
	push = true
	if file.Git == nil {
		return "", push
	}
	if file.Git.Remote != nil {
		remote = *file.Git.Remote
	}
	if file.Git.Push != nil {
		push = *file.Git.Push
	}
	return remote, push
}

// writeSyncReport prints what a sync did: whether it created a repository,
// whether it committed or found nothing to commit, and whether it pushed and to
// which remote. A push that was not requested prints nothing about pushing.
func writeSyncReport(out io.Writer, result gitx.Result, remote string) error {
	if result.Initialised {
		if _, err := fmt.Fprintln(out, "initialised git repository"); err != nil {
			return err
		}
	}
	outcome := "nothing to commit"
	if result.Committed {
		outcome = "committed changes"
	}
	if _, err := fmt.Fprintln(out, outcome); err != nil {
		return err
	}
	if result.Pushed {
		if remote == "" {
			remote = "origin"
		}
		if _, err := fmt.Fprintln(out, "pushed to "+remote); err != nil {
			return err
		}
	}
	return nil
}
