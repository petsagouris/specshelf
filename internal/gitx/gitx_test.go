package gitx_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/petsagouris/specshelf/internal/gitx"
)

// newGitTest skips the test when git is not installed and isolates it from the
// machine's git configuration. The identity comes from the environment and the
// global and system config files are empty, so no developer setting — a global
// identity, a hook, init.defaultBranch — can change the outcome, and a machine
// whose git has no identity still commits.
func newGitTest(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not found on PATH: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "Specshelf Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Specshelf Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
}

// git runs a git command in dir and returns its trimmed output, failing the test
// on a non-zero exit.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// write writes one file, creating its directory.
func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// bareRemote creates an empty bare repository to push to, never a network one.
func bareRemote(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "--bare", "-q")
	return dir
}

// TestSyncCommitsAndPushes is the spec's "Commit and push with an existing
// remote": pending changes are committed and the branch reaches the remote.
func TestSyncCommitsAndPushes(t *testing.T) {
	newGitTest(t)
	remote := bareRemote(t)
	root := t.TempDir()
	git(t, root, "init", "-q")
	git(t, root, "remote", "add", "origin", remote)
	write(t, root, "spec.md", "hello")

	result, err := gitx.Sync(gitx.Options{Root: root, Push: true, Remote: "origin"})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !result.Committed || !result.Pushed {
		t.Fatalf("result = %+v, want Committed and Pushed", result)
	}
	if status := git(t, root, "status", "--porcelain"); status != "" {
		t.Fatalf("working tree not clean after commit: %q", status)
	}
	branch := git(t, root, "symbolic-ref", "--short", "HEAD")
	local := git(t, root, "rev-parse", "HEAD")
	if got := git(t, remote, "rev-parse", "refs/heads/"+branch); got != local {
		t.Fatalf("remote refs/heads/%s = %s, want %s", branch, got, local)
	}
}

// TestSyncInitialisesRepositoryInPlainDirectory is "First sync in a plain
// directory": a repository is created in the vault root and the documents are
// committed.
func TestSyncInitialisesRepositoryInPlainDirectory(t *testing.T) {
	newGitTest(t)
	root := t.TempDir() // no .git
	write(t, root, "spec.md", "hello")

	result, err := gitx.Sync(gitx.Options{Root: root})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !result.Initialised || !result.Committed {
		t.Fatalf("result = %+v, want Initialised and Committed", result)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Fatalf("no repository in the vault root: %v", err)
	}
	if status := git(t, root, "status", "--porcelain"); status != "" {
		t.Fatalf("documents not committed: %q", status)
	}
}

// TestSyncRepeatedIsNothingToCommit is "Repeated sync": the second run reports
// nothing to commit — Committed false, no error — rather than git commit's
// non-zero status.
func TestSyncRepeatedIsNothingToCommit(t *testing.T) {
	newGitTest(t)
	root := t.TempDir()
	write(t, root, "spec.md", "hello")

	first, err := gitx.Sync(gitx.Options{Root: root})
	if err != nil || !first.Committed {
		t.Fatalf("first Sync = %+v, %v; want a commit", first, err)
	}
	second, err := gitx.Sync(gitx.Options{Root: root})
	if err != nil {
		t.Fatalf("second Sync: %v", err)
	}
	if second.Committed {
		t.Fatalf("second Sync = %+v, want nothing to commit", second)
	}
	if count := git(t, root, "rev-list", "--count", "HEAD"); count != "1" {
		t.Fatalf("commit count = %s, want 1", count)
	}
}

// TestSyncNoPushLeavesRemoteUntouched is "Local-only sync": a commit is created
// locally and the remote sees nothing.
func TestSyncNoPushLeavesRemoteUntouched(t *testing.T) {
	newGitTest(t)
	remote := bareRemote(t)
	root := t.TempDir()
	git(t, root, "init", "-q")
	git(t, root, "remote", "add", "origin", remote)
	write(t, root, "spec.md", "hello")

	result, err := gitx.Sync(gitx.Options{Root: root, Push: false})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !result.Committed || result.Pushed {
		t.Fatalf("result = %+v, want Committed and not Pushed", result)
	}
	if refs := git(t, remote, "for-each-ref", "--format=%(refname)", "refs/heads"); refs != "" {
		t.Fatalf("remote received refs despite --no-push: %s", refs)
	}
}

// TestSyncRejectedPushIsAnError is "Rejected push": a diverged remote makes the
// push fail, the error names the push, and neither local nor remote history is
// rewritten.
func TestSyncRejectedPushIsAnError(t *testing.T) {
	newGitTest(t)

	// A bare remote with a seed commit, and two clones of it.
	seed := t.TempDir()
	git(t, seed, "init", "-q")
	write(t, seed, "f.md", "seed")
	git(t, seed, "add", "-A")
	git(t, seed, "commit", "-qm", "seed")
	remote := t.TempDir()
	git(t, seed, "clone", "--bare", ".", remote)
	root := t.TempDir()
	git(t, seed, "clone", remote, root)
	other := t.TempDir()
	git(t, seed, "clone", remote, other)

	// The other clone moves the remote ahead, so root's push cannot fast-forward.
	write(t, other, "other.md", "other")
	git(t, other, "add", "-A")
	git(t, other, "commit", "-qm", "other")
	git(t, other, "push", "origin", "HEAD")

	write(t, root, "spec.md", "hello")
	remoteBefore := git(t, remote, "rev-list", "--count", "--all")

	result, err := gitx.Sync(gitx.Options{Root: root, Push: true, Remote: "origin"})
	if err == nil {
		t.Fatalf("Sync succeeded against a diverged remote; result = %+v", result)
	}
	if !strings.Contains(err.Error(), "push to origin") {
		t.Fatalf("error does not name the push: %v", err)
	}
	if !result.Committed {
		t.Fatalf("result = %+v, want the local commit to have happened", result)
	}
	if after := git(t, remote, "rev-list", "--count", "--all"); after != remoteBefore {
		t.Fatalf("remote history changed: %s -> %s", remoteBefore, after)
	}
	if count := git(t, root, "rev-list", "--count", "HEAD"); count != "2" {
		t.Fatalf("local commit count = %s, want 2 (seed + our commit, nothing rewritten)", count)
	}
}

// TestSyncMissingGitIsAClearError is "git not installed": the error is the
// package's sentinel and names the missing executable, and nothing else about the
// vault is touched.
func TestSyncMissingGitIsAClearError(t *testing.T) {
	// Deliberately no skip: an empty PATH makes git missing on any machine.
	t.Setenv("PATH", t.TempDir())

	_, err := gitx.Sync(gitx.Options{Root: t.TempDir()})
	if !errors.Is(err, gitx.ErrGitMissing) {
		t.Fatalf("err = %v, want ErrGitMissing", err)
	}
	if !strings.Contains(err.Error(), "git") {
		t.Fatalf("error does not name the missing executable: %v", err)
	}
}

// TestSyncVaultInsideAnotherRepositoryGetsItsOwn covers the edge a dotfiles tree
// creates: a vault inside someone else's repository is initialised as its own,
// so the documents are never committed into an unrelated history (D8).
func TestSyncVaultInsideAnotherRepositoryGetsItsOwn(t *testing.T) {
	newGitTest(t)
	outer := t.TempDir()
	git(t, outer, "init", "-q")
	root := filepath.Join(outer, "vault")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "spec.md", "hello")

	result, err := gitx.Sync(gitx.Options{Root: root})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if !result.Initialised {
		t.Fatalf("result = %+v, want the vault to get its own repository", result)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Fatalf("no repository in the vault root: %v", err)
	}
}
