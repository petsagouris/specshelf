package gitx

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrGitMissing reports that the git executable is not on PATH. Git is needed by
// sync and by no other command, which is what the error's message says.
var ErrGitMissing = errors.New("git executable not found on PATH")

// commitMessage is the one-line message every sync commit carries.
const commitMessage = "specshelf sync"

// Options is one sync request, built by a command from its flags and from the
// configuration file.
type Options struct {
	// Root is the vault directory, used as the working directory of every git
	// command.
	Root string
	// Push pushes the branch to Remote once the commit is made.
	Push bool
	// Remote is the git remote to push to. Empty means "origin".
	Remote string
}

// Result reports what a sync did, so a command can print it. Committed false is
// the nothing-to-commit success, not an error.
type Result struct {
	// Initialised reports that the vault was not a repository and one was created.
	Initialised bool
	// Committed reports that a commit was created.
	Committed bool
	// Pushed reports that the branch was pushed to the remote.
	Pushed bool
}

// Sync stages every change under opts.Root, commits it and — when opts.Push is
// set — pushes the branch to opts.Remote, all by running the git executable with
// the vault as its working directory, so the user's configuration, credential
// helper, SSH agent and hooks apply (D9). A vault that is not a repository is
// initialised first. With nothing to commit it returns a successful Result whose
// Committed is false rather than the commit command's non-zero status. A failed
// push returns the Result so far — the local commit did happen — with an error
// carrying git's own explanation; local and remote history are left alone.
func Sync(opts Options) (Result, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return Result{}, fmt.Errorf("%w: sync needs git; no other specshelf command does", ErrGitMissing)
	}

	var result Result
	if !isRepository(opts.Root) {
		if _, err := runGit(opts.Root, "init"); err != nil {
			return Result{}, fmt.Errorf("initialise git repository in %s: %w", opts.Root, err)
		}
		result.Initialised = true
	}
	if _, err := runGit(opts.Root, "add", "-A"); err != nil {
		return Result{}, fmt.Errorf("stage vault changes: %w", err)
	}
	status, err := runGit(opts.Root, "status", "--porcelain")
	if err != nil {
		return Result{}, fmt.Errorf("inspect vault changes: %w", err)
	}
	if strings.TrimSpace(string(status)) != "" {
		if _, err := runGit(opts.Root, "commit", "-m", commitMessage); err != nil {
			return Result{}, fmt.Errorf("commit vault changes: %w", err)
		}
		result.Committed = true
	}
	if opts.Push && hasCommit(opts.Root) {
		remote := opts.Remote
		if remote == "" {
			remote = "origin"
		}
		// HEAD rather than a branch name: it pushes the current branch under its
		// own name and needs no upstream to have been configured.
		if _, err := runGit(opts.Root, "push", remote, "HEAD"); err != nil {
			return result, fmt.Errorf("push to %s: %w", remote, err)
		}
		result.Pushed = true
	}
	return result, nil
}

// isRepository reports whether root is itself the top level of a git repository.
// A vault inside someone else's repository — a dotfiles tree, say — is not one:
// committing the vault into it would drop the user's documents into an unrelated
// history, so such a vault gets its own repository (D8).
func isRepository(root string) bool {
	out, err := runGit(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return false
	}
	return samePath(strings.TrimSpace(string(out)), root)
}

// hasCommit reports whether HEAD resolves, i.e. the repository has a commit to
// push. A repository initialised with nothing committed has none, and that is the
// nothing-to-commit success rather than a push failure.
func hasCommit(root string) bool {
	_, err := runGit(root, "rev-parse", "--verify", "--quiet", "HEAD")
	return err == nil
}

// samePath compares two paths by their resolved, cleaned form.
func samePath(a, b string) bool {
	if resolved, err := filepath.EvalSymlinks(a); err == nil {
		a = resolved
	}
	if resolved, err := filepath.EvalSymlinks(b); err == nil {
		b = resolved
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

// runGit runs git in dir and returns its combined output. A non-zero exit keeps
// git's own message on the error, so the caller's message says what git said.
func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if message := strings.TrimSpace(string(out)); message != "" {
			return out, fmt.Errorf("%s: %w", message, err)
		}
		return out, err
	}
	return out, nil
}
