package tui

import (
	"errors"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
)

// NotTerminalError reports that the session cannot start because its output is
// not an interactive terminal. The message points the user at the
// non-interactive commands.
type NotTerminalError struct{}

func (NotTerminalError) Error() string {
	return `stdout is not a terminal; use "specshelf list", "specshelf search" or "specshelf show" for non-interactive output`
}

// config is the session's streams, taken from the process unless an option
// overrides them.
type config struct {
	input  io.Reader
	output io.Writer
}

// Option overrides a session stream, so tests can run without a terminal.
type Option func(*config)

// WithInput sets the session's input stream.
func WithInput(input io.Reader) Option { return func(c *config) { c.input = input } }

// WithOutput sets the session's output stream.
func WithOutput(output io.Writer) Option { return func(c *config) { c.output = output } }

func newConfig(opts []Option) config {
	c := config{input: os.Stdin, output: os.Stdout}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// stdoutIsTerminal reports whether w is an interactive terminal. It is a
// variable so a test can exercise the guard without a real terminal.
var stdoutIsTerminal = func(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// Run starts the interactive session against store, reading stdin and writing
// stdout. It fails with a NotTerminalError before the store is touched when the
// output is not a terminal, so a redirected invocation exits without starting a
// session or modifying the vault.
func Run(store Store, opts ...Option) error {
	cfg := newConfig(opts)
	if !stdoutIsTerminal(cfg.output) {
		return NotTerminalError{}
	}
	return run(store, cfg)
}

// run starts the program and returns when the session ends for any reason. The
// terminal is restored by the program on every exit path — quit, error, a
// terminal that went away and an input stream that closed.
func run(store Store, cfg config) error {
	watcher := &inputWatcher{reader: cfg.input}
	program := tea.NewProgram(New(store), tea.WithInput(watcher), tea.WithOutput(cfg.output))
	watcher.quit = program.Quit
	_, err := program.Run()
	return err
}

// inputWatcher ends the session when the input stream closes. The program's
// reader treats end-of-input as a silent stop, which would otherwise leave the
// session waiting for a message that never comes, so EOF is turned into a quit.
type inputWatcher struct {
	reader io.Reader
	quit   func()
}

func (w *inputWatcher) Read(p []byte) (int, error) {
	n, err := w.reader.Read(p)
	if errors.Is(err, io.EOF) && w.quit != nil {
		w.quit()
	}
	return n, err
}
