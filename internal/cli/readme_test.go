package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadmeQuickstart runs the command sequence the README documents, against
// a temporary vault, so a documented command or flag that no longer exists
// fails the build. The sequence is derived from the README rather than copied
// into Go: editing the README is what changes the test.
func TestReadmeQuickstart(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	commands := quickstartCommands(t, string(data))
	if len(commands) == 0 {
		t.Fatal("the README Quickstart section documents no commands")
	}

	// The default vault lives under HOME, so a temporary home keeps the real
	// vault out of reach; the config path is isolated for the same reason.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SPECSHELF_CONFIG", filepath.Join(home, "config.toml"))

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = home
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("documented command %q failed: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
}

// quickstartCommands extracts the commands from the fenced block that follows
// the README's "## Quickstart" heading. Blank lines and comments are skipped;
// every other line must be a specshelf command, which is what makes a renamed
// command fail here rather than silently disappearing.
func quickstartCommands(t *testing.T, readme string) [][]string {
	t.Helper()
	lines := strings.Split(readme, "\n")

	i := 0
	for ; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "## Quickstart" {
			break
		}
	}
	if i == len(lines) {
		t.Fatal("the README has no '## Quickstart' section")
	}
	for ; i < len(lines) && !strings.HasPrefix(lines[i], "```"); i++ {
	}
	if i == len(lines) {
		t.Fatal("the Quickstart section has no fenced code block")
	}

	var commands [][]string
	for i++; i < len(lines) && !strings.HasPrefix(lines[i], "```"); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if fields[0] != "specshelf" {
			t.Fatalf("quickstart line %q is not a specshelf command", line)
		}
		commands = append(commands, fields)
	}
	return commands
}
