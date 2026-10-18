package cli

import (
	"fmt"
	"io"
	"sort"

	"github.com/petsagouris/specshelf/internal/vault"
	"github.com/spf13/cobra"
)

// newDoctorCommand validates the vault and reports every problem. The findings
// are diagnostics, so they go to stderr with the specshelf prefix and stdout
// stays empty; the command returns an error when there is at least one so the
// exit code is non-zero, and succeeds silently for a healthy vault.
//
// Every finding the store reports is listed, one line each — the missing
// layout, the unreadable documents, the kind mismatches and the identity
// mismatches — so the count in the final error is never the only trace of a
// problem. The checks themselves live in vault.Validate; this command prints
// what the store reports and adds no check of its own, so a second definition
// of the vault's rules cannot drift from the store's.
func newDoctorCommand() *cobra.Command {
	var rebuildIndex bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check the vault and report every problem.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			report, err := vault.Validate(resolution.Root)
			if err != nil {
				return err
			}
			for _, missing := range report.Missing {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "specshelf: %s\n", missing.Text())
			}
			printWarnings(cmd, report.Unreadable)
			for _, mismatch := range report.Mismatches {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "specshelf: %s\n", mismatch.Text())
			}
			for _, identity := range report.Identities {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "specshelf: %s\n", identity.Text())
			}
			if rebuildIndex {
				if err := vault.WriteIndex(resolution.Root); err != nil {
					return err
				}
			}
			if problems := report.Problems(); problems > 0 {
				if problems == 1 {
					return fmt.Errorf("1 problem found")
				}
				return fmt.Errorf("%d problems found", problems)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&rebuildIndex, "index", false, "Regenerate index.jsonl after the report.")
	return cmd
}

// newStatsCommand prints the vault's document counts grouped by kind, status,
// repo and tag. The counts are the payload, so they go to stdout as three
// tab-separated columns — group, key, count — one per line, groups in a fixed
// order and keys sorted, so the output is stable.
func newStatsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Report document counts by kind, status, repo and tag.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			stats, warnings, err := vault.Stats(resolution.Root)
			if err != nil {
				return err
			}
			printWarnings(cmd, warnings)
			return writeStats(cmd.OutOrStdout(), stats)
		},
	}
}

// writeStats renders the grouped counts as TSV. Each group's keys are sorted so
// two runs over the same vault print the same bytes.
func writeStats(out io.Writer, stats vault.Statistics) error {
	groups := []struct {
		group  string
		counts map[string]int
	}{
		{"kind", stats.ByKind},
		{"status", stats.ByStatus},
		{"repo", stats.ByRepo},
		{"tag", stats.ByTag},
	}
	for _, g := range groups {
		keys := make([]string, 0, len(g.counts))
		for key := range g.counts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if _, err := fmt.Fprintf(out, "%s\t%s\t%d\n", g.group, key, g.counts[key]); err != nil {
				return err
			}
		}
	}
	return nil
}
