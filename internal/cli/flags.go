package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/petsagouris/specshelf/internal/vault"
	"github.com/spf13/cobra"
)

// resolveVault reads the vault selectors off the command and resolves the one
// vault it operates on. Any warning the configuration file produced is written
// to stderr, prefixed like every other line the command surface emits.
func resolveVault(cmd *cobra.Command) (vault.Resolution, error) {
	flags := cmd.Flags()
	home, err := flags.GetString("home")
	if err != nil {
		return vault.Resolution{}, err
	}
	name, err := flags.GetString("vault")
	if err != nil {
		return vault.Resolution{}, err
	}
	project, err := flags.GetBool("project")
	if err != nil {
		return vault.Resolution{}, err
	}
	dir, err := os.Getwd()
	if err != nil {
		return vault.Resolution{}, err
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return vault.Resolution{}, err
	}

	resolution, err := vault.Resolve(vault.Selectors{Home: home, Vault: name, Project: project, Dir: dir}, userHome)
	if err != nil {
		return vault.Resolution{}, err
	}
	for _, warning := range resolution.Warnings {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "specshelf: %s\n", warning)
	}
	return resolution, nil
}

// stringList collects the raw values of a flag and its alias in the order they
// are given. pflag's StringArray resets on the first Set of each flag, so two
// aliases sharing one destination would lose the interleaving the reference
// tool's argparse dest preserves; this Value always appends.
type stringList []string

func (s *stringList) String() string         { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error { *s = append(*s, value); return nil }
func (s *stringList) Type() string           { return "value" }

// splitValues merges the raw values of one flag's aliases into the reference
// tool's list: comma-separated parts and repeated flags alike, each trimmed,
// empty parts dropped, de-duplicated in first-seen order.
func splitValues(groups ...[]string) []string {
	var out []string
	seen := make(map[string]bool)
	for _, group := range groups {
		for _, raw := range group {
			for _, part := range strings.Split(raw, ",") {
				value := strings.TrimSpace(part)
				if value == "" || seen[value] {
					continue
				}
				seen[value] = true
				out = append(out, value)
			}
		}
	}
	return out
}

// sessionValues adds the current Codex session when asked, reading
// CODEX_THREAD_ID then CODEX_SESSION_ID, and errors naming the variable when
// neither is set.
func sessionValues(raw []string, current bool) ([]string, error) {
	sessions := splitValues(raw)
	if !current {
		return sessions, nil
	}
	id := os.Getenv("CODEX_THREAD_ID")
	if id == "" {
		id = os.Getenv("CODEX_SESSION_ID")
	}
	if id == "" {
		return nil, fmt.Errorf("CODEX_THREAD_ID is not set")
	}
	return splitValues(sessions, []string{id}), nil
}
