package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/petsagouris/specshelf/internal/config"
)

// Source names the selector level that chose a vault, so a caller can report
// where the resolved vault came from.
type Source string

const (
	// SourceHome is the --home flag.
	SourceHome Source = "--home"
	// SourceEnv is the SPECSHELF_HOME environment variable.
	SourceEnv Source = "SPECSHELF_HOME"
	// SourceVault is the --vault named vault selector.
	SourceVault Source = "--vault"
	// SourceProject is the --project marker search.
	SourceProject Source = "--project"
	// SourceConfig is the configuration file's default vault.
	SourceConfig Source = "config"
	// SourceDefault is the built-in ~/.agents/spec-vault.
	SourceDefault Source = "default"
)

// Selectors are the vault selectors a command was invoked with. The zero value
// selects no vault explicitly.
type Selectors struct {
	Home    string // --home
	Vault   string // --vault
	Project bool   // --project
	Dir     string // working directory, searched only by --project
}

// Resolution is a resolved vault: the root a command operates on, the selector
// level that chose it, and the warnings the configuration file produced.
type Resolution struct {
	Root     string
	Source   Source
	Warnings []string
}

// ConflictError reports two explicit vault selectors used together. 3.7 turns it
// into the usage exit code.
type ConflictError struct {
	First, Second string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%s and %s are mutually exclusive", e.First, e.Second)
}

// UnknownVaultError reports a --vault name the configuration does not declare.
type UnknownVaultError struct {
	Name     string
	Declared []string
}

func (e *UnknownVaultError) Error() string {
	if len(e.Declared) == 0 {
		return fmt.Sprintf("unknown vault %q: no vaults are declared", e.Name)
	}
	return fmt.Sprintf("unknown vault %q: declared vaults are %s", e.Name, strings.Join(e.Declared, ", "))
}

// NoProjectVaultError reports a --project search that found no .specshelf marker.
type NoProjectVaultError struct {
	Start string
}

func (e *NoProjectVaultError) Error() string {
	return fmt.Sprintf("no project vault: no .specshelf directory found from %s", e.Start)
}

// defaultVault is the vault used when nothing else selects one, relative to home.
const defaultVault = ".agents/spec-vault"

// Resolve returns the vault the selectors select and the level that selected it,
// using the first of: --home, SPECSHELF_HOME, --vault through the configuration
// file, --project, the configuration file's default vault, and
// ~/.agents/spec-vault. home is the user's home directory and bounds the project
// walk; the configuration file is read from config.Path(). Two explicit
// selectors are a ConflictError.
func Resolve(sel Selectors, home string) (Resolution, error) {
	if err := checkConflicts(sel); err != nil {
		return Resolution{}, err
	}
	path, err := config.Path()
	if err != nil {
		return Resolution{}, err
	}
	file, warnings, err := config.Load(path)
	if err != nil {
		return Resolution{}, err
	}
	res := Resolution{Warnings: warnings}

	switch {
	case sel.Home != "":
		res.Root, res.Source = sel.Home, SourceHome
	case os.Getenv("SPECSHELF_HOME") != "":
		res.Root, res.Source = os.Getenv("SPECSHELF_HOME"), SourceEnv
	case sel.Vault != "":
		root, err := namedVault(file, sel.Vault, home)
		if err != nil {
			return Resolution{}, err
		}
		res.Root, res.Source = root, SourceVault
	case sel.Project:
		root, err := Discover(sel.Dir, home)
		if err != nil {
			return Resolution{}, err
		}
		res.Root, res.Source = root, SourceProject
	case file.DefaultVault != nil && *file.DefaultVault != "":
		res.Root, res.Source = expandHome(*file.DefaultVault, home), SourceConfig
	default:
		res.Root, res.Source = filepath.Join(home, defaultVault), SourceDefault
	}
	return res, nil
}

// checkConflicts rejects two explicit selectors used together, naming both so
// the message can be turned into a usage error.
func checkConflicts(sel Selectors) error {
	switch {
	case sel.Home != "" && sel.Vault != "":
		return &ConflictError{First: "--home", Second: "--vault"}
	case sel.Home != "" && sel.Project:
		return &ConflictError{First: "--home", Second: "--project"}
	case sel.Vault != "" && sel.Project:
		return &ConflictError{First: "--vault", Second: "--project"}
	}
	return nil
}

// namedVault returns the path of a declared named vault. An undeclared name, or
// a declared one without a path, is an UnknownVaultError listing the names.
func namedVault(file *config.File, name, home string) (string, error) {
	if v, ok := file.Vaults[name]; ok && v.Path != "" {
		return expandHome(v.Path, home), nil
	}
	declared := make([]string, 0, len(file.Vaults))
	for n := range file.Vaults {
		declared = append(declared, n)
	}
	sort.Strings(declared)
	return "", &UnknownVaultError{Name: name, Declared: declared}
}

// expandHome resolves a leading "~" in a configured path against home.
func expandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~"+string(filepath.Separator)) {
		return filepath.Join(home, path[2:])
	}
	return path
}
