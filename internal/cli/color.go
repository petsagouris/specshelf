package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/petsagouris/specshelf/internal/config"
	"github.com/spf13/cobra"
)

// The three --color modes. auto colours only a terminal with NO_COLOR unset and
// TERM other than dumb; always and never say so outright.
const (
	colorAuto   = "auto"
	colorAlways = "always"
	colorNever  = "never"
)

// colorModeValue is the --color flag. It validates as it parses, so an invalid
// mode is a usage error (exit 2) wherever the flag is accepted.
type colorModeValue string

func (c *colorModeValue) String() string { return string(*c) }
func (c *colorModeValue) Type() string   { return "string" }

func (c *colorModeValue) Set(value string) error {
	switch value {
	case colorAuto, colorAlways, colorNever:
		*c = colorModeValue(value)
		return nil
	}
	// pflag wraps this in `invalid argument %q for %q flag: %v`, so the reason
	// alone belongs here.
	return errors.New("must be auto, always, or never")
}

// resolveColor is the effective colour mode: the --color flag, then the
// configuration file's color setting, then auto. A configuration value outside
// the three modes is an error rather than a silent fallback to auto.
func resolveColor(cmd *cobra.Command) (string, error) {
	flags := cmd.Flags()
	if flags.Changed("color") {
		return flags.GetString("color")
	}
	path, err := config.Path()
	if err != nil {
		return "", err
	}
	file, _, err := config.Load(path)
	if err != nil {
		return "", err
	}
	if file.Color == nil {
		return colorAuto, nil
	}
	switch *file.Color {
	case colorAuto, colorAlways, colorNever:
		return *file.Color, nil
	}
	return "", fmt.Errorf("invalid color mode %q in %s: must be auto, always, or never", *file.Color, path)
}

// isTerminal reports whether w is a character device. It is a variable so a test
// can stand in a terminal, which a test process does not have.
var isTerminal = func(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// paintEnabled reports whether a display command may write escapes. A machine
// payload (--json, or any --format but table) never colours, whatever the mode
// says, because callers parse it; otherwise the mode decides, and auto needs a
// terminal, NO_COLOR unset and TERM other than dumb.
func paintEnabled(mode string, out io.Writer, machine bool) bool {
	if machine {
		return false
	}
	switch mode {
	case colorAlways:
		return true
	case colorNever:
		return false
	}
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal(out)
}

// style writes the escape sequences a display command may carry. A disabled
// style returns its input unchanged, so one renderer serves every mode.
type style struct{ on bool }

func (s style) bold(text string) string {
	if !s.on {
		return text
	}
	return "\x1b[1m" + text + "\x1b[0m"
}

// displayStyle resolves the style for a display command's stdout. machine is
// true for --json and for any --format but table.
func displayStyle(cmd *cobra.Command, machine bool) (style, error) {
	mode, err := resolveColor(cmd)
	if err != nil {
		return style{}, err
	}
	return style{on: paintEnabled(mode, cmd.OutOrStdout(), machine)}, nil
}
