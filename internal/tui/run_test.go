package tui

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// restoreSequences are what the renderer emits on the way out: leave the
// alternate screen and show the cursor again.
var restoreSequences = []string{"\x1b[?1049l", "\x1b[?25h"}

func assertRestored(t *testing.T, output string) {
	t.Helper()
	for _, want := range restoreSequences {
		if !strings.Contains(output, want) {
			t.Fatalf("terminal not restored: output does not contain %q\n%q", want, output)
		}
	}
}

func TestRunRejectsNonTerminal(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	store := &fakeStore{records: sampleRecords()}

	err = Run(store, WithOutput(file))
	var notTerminal NotTerminalError
	if !errors.As(err, &notTerminal) {
		t.Fatalf("Run with a redirected output returned %v, want a NotTerminalError", err)
	}
	if store.lists != 0 {
		t.Fatalf("the guard touched the vault: %d List calls", store.lists)
	}
}

func TestRunRejectsNonFileOutput(t *testing.T) {
	var buf bytes.Buffer
	if err := Run(&fakeStore{}, WithOutput(&buf)); !errors.As(err, new(NotTerminalError)) {
		t.Fatalf("Run with a buffer output returned %v, want a NotTerminalError", err)
	}
}

func TestQuitRestoresTerminal(t *testing.T) {
	var output bytes.Buffer
	err := run(&fakeStore{records: sampleRecords()}, config{
		input:  strings.NewReader("q"),
		output: &output,
	})
	if err != nil {
		t.Fatalf("run returned %v, want nil", err)
	}
	assertRestored(t, output.String())
}

func TestClosedInputEndsSession(t *testing.T) {
	var output bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- run(&fakeStore{records: sampleRecords()}, config{
			input:  strings.NewReader(""),
			output: &output,
		})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned %v, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the session did not end when its input closed")
	}
	assertRestored(t, output.String())
}

func TestClosedTerminalQuits(t *testing.T) {
	m, _ := newFake()
	cmd := update(m, tea.WindowSizeMsg{})
	if !m.quitting {
		t.Fatal("a zero-sized terminal did not end the session")
	}
	if msg := execCmd(t, cmd); msg != tea.Quit() {
		t.Fatalf("zero-sized terminal produced %#v, want QuitMsg", msg)
	}
}
