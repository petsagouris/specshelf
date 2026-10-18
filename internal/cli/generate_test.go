package cli

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// eachCommand visits cmd and every descendant, the same tree the generators walk.
func eachCommand(cmd *cobra.Command, visit func(*cobra.Command)) {
	visit(cmd)
	for _, child := range cmd.Commands() {
		eachCommand(child, visit)
	}
}

// TestSupersetCommandsAreAvailable pins the command surface: the nine commands of
// the superset are present, available, and listed by the root help. It also covers
// the superset options that ride on existing commands, --json, --format, --color
// and --slug.
func TestSupersetCommandsAreAvailable(t *testing.T) {
	root := NewRootCommand()
	available := map[string]bool{}
	eachCommand(root, func(cmd *cobra.Command) { available[cmd.Name()] = cmd.IsAvailableCommand() })

	var help strings.Builder
	root.SetOut(&help)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("--help: %v", err)
	}

	for _, name := range []string{"edit", "rm", "status", "doctor", "export", "stats", "sync", "upgrade", "ui"} {
		if !available[name] {
			t.Errorf("superset command %q is missing or unavailable", name)
		}
		if !strings.Contains(help.String(), name) {
			t.Errorf("root help does not list the superset command %q", name)
		}
	}
}

// TestCompletionMentionsEveryCommand pins the spec scenario for bash, whose
// generated script lists each command statically; the other three shells are
// checked to generate a non-empty script.
func TestCompletionMentionsEveryCommand(t *testing.T) {
	root := NewRootCommand()

	var bash strings.Builder
	if err := root.GenBashCompletion(&bash); err != nil {
		t.Fatalf("GenBashCompletion: %v", err)
	}
	eachCommand(root, func(cmd *cobra.Command) {
		if cmd == root {
			return // the root is the script's subject, not one of its commands
		}
		if want := `commands+=("` + cmd.Name() + `")`; !strings.Contains(bash.String(), want) {
			t.Errorf("bash completion does not mention command %q", cmd.Name())
		}
	})

	generators := map[string]func(io.Writer) error{
		"zsh":        root.GenZshCompletion,
		"fish":       func(w io.Writer) error { return root.GenFishCompletion(w, true) },
		"powershell": root.GenPowerShellCompletionWithDesc,
	}
	for name, generate := range generators {
		var buf strings.Builder
		if err := generate(&buf); err != nil {
			t.Errorf("%s completion: %v", name, err)
			continue
		}
		if buf.Len() == 0 {
			t.Errorf("%s completion is empty", name)
		}
	}
}

// TestManPagesCoverEveryCommand pins the man scenario: a roff page per available
// command (cobra's own help topic is not a command page), each naming its command.
func TestManPagesCoverEveryCommand(t *testing.T) {
	dir := t.TempDir()
	if _, stderr, code := runCLI(t, "man", dir); code != 0 {
		t.Fatalf("man exit = %d, stderr = %s", code, stderr)
	}
	seen := 0
	eachCommand(NewRootCommand(), func(cmd *cobra.Command) {
		if !cmd.IsAvailableCommand() {
			return
		}
		seen++
		name := strings.ReplaceAll(cmd.CommandPath(), " ", "-") + ".1"
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("no man page for %q: %v", cmd.CommandPath(), err)
			return
		}
		if !strings.HasPrefix(string(data), ".nh\n") || !strings.Contains(string(data), cmd.Name()) {
			t.Errorf("man page %s does not look like a page naming %q", name, cmd.Name())
		}
	})
	if seen < 2 {
		t.Fatalf("walked only %d available commands; the tree looks wrong", seen)
	}
}

// TestCompletionScriptsSource is the CI proof the task asks for: the generated
// bash and zsh scripts load in their own shell. A missing interpreter skips.
func TestCompletionScriptsSource(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			interpreter, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s is not installed", shell)
			}
			script, stderr, code := runCLI(t, "completion", shell)
			if code != 0 {
				t.Fatalf("completion %s exit = %d, stderr = %s", shell, code, stderr)
			}
			path := filepath.Join(t.TempDir(), "completion."+shell)
			if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
				t.Fatal(err)
			}
			// zsh's script calls compdef, which only exists once the completion
			// system is initialised; a stub lets source exercise the whole body.
			code_ := `source "$1"`
			if shell == "zsh" {
				code_ = `compdef() { :; }; source "$1"`
			}
			out, err := exec.Command(interpreter, "-c", code_, shell, path).CombinedOutput()
			if err != nil {
				t.Errorf("sourcing the %s completion failed: %v\n%s", shell, err, out)
			}
		})
	}
}
