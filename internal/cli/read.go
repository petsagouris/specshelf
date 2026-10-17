package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/petsagouris/specshelf/internal/vault"
	"github.com/spf13/cobra"
)

// usagef builds a usage error, which exits 2. It is used where a flag's value is
// rejected after cobra has parsed it.
func usagef(format string, args ...any) error {
	return &usageError{err: fmt.Errorf(format, args...)}
}

// addFilterFlags registers the filter flags shared by list, search and prime.
func addFilterFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.String("kind", "all", "Only documents of this kind: all, spec, or decision.")
	flags.String("repo", "", "Only documents carrying this repo.")
	flags.String("tag", "", "Only documents carrying this tag.")
	flags.String("codex-session", "", "Only documents linked to this session.")
	flags.String("status", "", "Only documents with this status.")
}

// filterValues reads the filter flags into the store's Filters, rejecting a kind
// outside the reference tool's three values as a usage error.
func filterValues(cmd *cobra.Command) (vault.Filters, error) {
	flags := cmd.Flags()
	kind, err := flags.GetString("kind")
	if err != nil {
		return vault.Filters{}, err
	}
	switch kind {
	case "all", "spec", "decision":
	default:
		return vault.Filters{}, usagef("invalid argument %q for \"--kind\" flag: must be all, spec, or decision", kind)
	}
	repo, err := flags.GetString("repo")
	if err != nil {
		return vault.Filters{}, err
	}
	tag, err := flags.GetString("tag")
	if err != nil {
		return vault.Filters{}, err
	}
	session, err := flags.GetString("codex-session")
	if err != nil {
		return vault.Filters{}, err
	}
	status, err := flags.GetString("status")
	if err != nil {
		return vault.Filters{}, err
	}
	return vault.Filters{Kind: kind, Repo: repo, Tag: tag, CodexSession: session, Status: status}, nil
}

// printWarnings writes one stderr line per skipped document.
func printWarnings(cmd *cobra.Command, warnings []vault.Warning) {
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "specshelf: %s\n", warning.Text())
	}
}

// newListCommand lists documents as the reference tool's nine-column TSV (the
// default), or in the table, csv or jsonl format the --format flag selects.
func newListCommand() *cobra.Command {
	var limit int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List specs and decisions.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			format, err := checkFormat(cmd, jsonOut)
			if err != nil {
				return err
			}
			filters, err := filterValues(cmd)
			if err != nil {
				return err
			}
			records, warnings, err := vault.Query(resolution.Root, filters)
			if err != nil {
				return err
			}
			printWarnings(cmd, warnings)
			records = limitRecords(records, limit)
			return writeList(cmd, format, records)
		},
	}

	addFilterFlags(cmd)
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of documents to list.")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit one JSON object per document, as --format jsonl does.")
	cmd.Flags().String("format", "tsv", "Output format: tsv, table, csv, or jsonl.")
	return cmd
}

// newSearchCommand prints one row per matching document: id, kind, vault relative
// path with its line, and the matching snippet — as TSV (the default) or in the
// table, csv or jsonl format the --format flag selects.
func newSearchCommand() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "search QUERY",
		Short: "Search specs and decisions.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			format, err := checkFormat(cmd, jsonOut)
			if err != nil {
				return err
			}
			filters, err := filterValues(cmd)
			if err != nil {
				return err
			}
			matches, warnings, err := vault.Search(resolution.Root, args[0], filters)
			if err != nil {
				return err
			}
			printWarnings(cmd, warnings)
			return writeSearch(cmd, format, matches)
		},
	}
	addFilterFlags(cmd)
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit one JSON object per match, as --format jsonl does.")
	cmd.Flags().String("format", "tsv", "Output format: tsv, table, csv, or jsonl.")
	return cmd
}

// newShowCommand prints one document's raw bytes, or its index record with
// --json.
func newShowCommand() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "show REF",
		Short: "Print one spec or decision.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			found, err := vault.Find(resolution.Root, args[0])
			if err != nil {
				return err
			}
			if !jsonOut {
				_, err = cmd.OutOrStdout().Write(found.Data)
				return err
			}
			record, err := indexRecordFor(resolution.Root, found.Path)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(vault.EncodeIndex([]vault.IndexRecord{record}))
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit the document's index record as JSON.")
	return cmd
}

// newPrimeCommand prints a Markdown summary of the matching documents.
func newPrimeCommand() *cobra.Command {
	var limit int
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "prime",
		Short: "Print an agent-friendly vault summary.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			filters, err := filterValues(cmd)
			if err != nil {
				return err
			}
			records, warnings, err := vault.Query(resolution.Root, filters)
			if err != nil {
				return err
			}
			printWarnings(cmd, warnings)
			records = limitRecords(records, limit)

			painter, err := displayStyle(cmd, jsonOut)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if jsonOut {
				_, err = out.Write(vault.EncodeIndex(records))
				return err
			}
			return writePrime(out, resolution.Root, records, painter)
		},
	}

	addFilterFlags(cmd)
	cmd.Flags().IntVar(&limit, "limit", 12, "Maximum number of documents to summarise.")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit JSONL records.")
	return cmd
}

// writePrime renders the reference tool's Markdown summary, colouring its
// headings when the caller's style is on.
func writePrime(out io.Writer, root string, records []vault.IndexRecord, painter style) error {
	if _, err := fmt.Fprintf(out, "%s\n\nVault: `%s`\n\n", painter.bold("# Agent Spec Vault"), root); err != nil {
		return err
	}
	if len(records) == 0 {
		_, err := fmt.Fprintln(out, "No matching specs or decisions found.")
		return err
	}
	if _, err := fmt.Fprint(out, painter.bold("## Relevant Specs And Decisions")+"\n\n"); err != nil {
		return err
	}
	for _, record := range records {
		if _, err := fmt.Fprintf(out, "- `%s` (%s, %s) %s\n  Path: `%s`\n  Repos: %s\n  Tags: %s\n  Codex sessions: %s\n",
			record.ID, record.Kind, orDash(record.Status), record.Title, record.Path,
			joinOr(record.Repos, ", ", "global"), joinOr(record.Tags, ", ", "-"), joinOr(record.CodexSessions, ", ", "-")); err != nil {
			return err
		}
	}
	return nil
}

// newPathCommand prints the resolved vault root, or the root and the selector
// level that chose it with --json (design D8).
func newPathCommand() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "path",
		Short: "Print the resolved vault directory.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			if jsonOut {
				return writePathJSON(cmd.OutOrStdout(), resolution)
			}
			_, err = cmd.OutOrStdout().Write([]byte(resolution.Root + "\n"))
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Emit the root and the selector that chose it.")
	return cmd
}

// limitRecords applies a command's --limit, treating a negative limit as zero,
// as the reference tool does.
func limitRecords(records []vault.IndexRecord, limit int) []vault.IndexRecord {
	if limit < 0 {
		limit = 0
	}
	if limit < len(records) {
		return records[:limit]
	}
	return records
}

// orDash renders an empty field as the reference tool's "-" placeholder.
func orDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

// joinOrDash joins a list field with commas, or "-" when it is empty.
func joinOrDash(items []string) string {
	return joinOr(items, ",", "-")
}

// joinOr joins items with sep, or returns empty when there are none.
func joinOr(items []string, sep, empty string) string {
	if len(items) == 0 {
		return empty
	}
	return strings.Join(items, sep)
}
