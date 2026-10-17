package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Source is the precedence level an effective setting value came from.
type Source int

const (
	// SourceDefault is the setting's built-in default.
	SourceDefault Source = iota
	// SourceConfig is the value from the configuration file.
	SourceConfig
	// SourceEnv is the value from an environment variable.
	SourceEnv
	// SourceFlag is the value from a command-line flag.
	SourceFlag
)

// String names the level, for reporting where an effective value came from.
func (s Source) String() string {
	switch s {
	case SourceFlag:
		return "flag"
	case SourceEnv:
		return "environment"
	case SourceConfig:
		return "config"
	default:
		return "default"
	}
}

// Resolve applies the precedence flag, then environment variable, then
// configuration file, then built-in default to one setting, and reports the
// level the value came from. A nil pointer means that level did not set the
// setting.
func Resolve[T any](flag, env, file *T, def T) (T, Source) {
	switch {
	case flag != nil:
		return *flag, SourceFlag
	case env != nil:
		return *env, SourceEnv
	case file != nil:
		return *file, SourceConfig
	default:
		return def, SourceDefault
	}
}

// File is the decoded configuration file. A nil field means the file did not set
// that setting, so a level before it applies. Vaults holds the named vaults
// declared with [vaults.<name>] sections.
type File struct {
	DefaultVault *string          `toml:"default_vault"`
	Color        *string          `toml:"color"`
	Editor       *string          `toml:"editor"`
	AutoIndex    *bool            `toml:"auto_index"`
	Project      *bool            `toml:"project"`
	Git          *Git             `toml:"git"`
	Vaults       map[string]Vault `toml:"vaults"`
}

// Git is the [git] section: the remote to push to and whether to push at all.
type Git struct {
	Remote *string `toml:"remote"`
	Push   *bool   `toml:"push"`
}

// Vault is one named vault from a [vaults.<name>] section.
type Vault struct {
	Path string `toml:"path"`
}

// Path reports the configuration file the tool reads: the file named by
// SPECSHELF_CONFIG when that variable is set and non-empty, and otherwise
// config.toml under the platform user configuration directory.
func Path() (string, error) {
	if path := os.Getenv("SPECSHELF_CONFIG"); path != "" {
		return path, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}
	return filepath.Join(dir, "specshelf", "config.toml"), nil
}

// Load reads the configuration file at path. A file that does not exist is not
// an error: Load reports an empty File. A file that cannot be read, is not valid
// TOML, or holds a wrongly-typed value returns an error naming path and the
// position of the problem. Unknown keys are returned as one warning each and do
// not stop the load.
func Load(path string) (*File, []string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &File{}, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	return Parse(path, data)
}

// Parse decodes TOML data into a File. name is used to locate a problem in error
// messages. Unknown keys are returned as warnings naming them; malformed input
// and wrongly-typed values return an error naming name and the line and column
// of the problem.
func Parse(name string, data []byte) (*File, []string, error) {
	file := &File{}
	decoder := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
	if err := decoder.Decode(file); err != nil {
		var unknown *toml.StrictMissingError
		if errors.As(err, &unknown) {
			return file, unknownKeyWarnings(unknown), nil
		}
		return nil, nil, locatedError(name, err)
	}
	return file, nil, nil
}

func unknownKeyWarnings(err *toml.StrictMissingError) []string {
	warnings := make([]string, 0, len(err.Errors))
	for i := range err.Errors {
		warnings = append(warnings, fmt.Sprintf("unknown configuration key %q", strings.Join(err.Errors[i].Key(), ".")))
	}
	return warnings
}

// locatedError renders a decode failure as file:line:column: message so the
// malformed location is visible; anything else is prefixed with the file name.
func locatedError(name string, err error) error {
	var decode *toml.DecodeError
	if errors.As(err, &decode) {
		line, column := decode.Position()
		message := strings.TrimPrefix(decode.Error(), "toml: ")
		return fmt.Errorf("%s:%d:%d: %s", name, line, column, message)
	}
	if name != "" {
		return fmt.Errorf("%s: %w", name, err)
	}
	return err
}
