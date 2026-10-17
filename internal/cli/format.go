package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/petsagouris/specshelf/internal/vault"
	"github.com/spf13/cobra"
)

// The --format values list and search accept. tsv is the reference default and
// the byte shape its callers already parse; table is the only human format and
// the only one colour may reach.
const (
	formatTSV   = "tsv"
	formatTable = "table"
	formatCSV   = "csv"
	formatJSONL = "jsonl"
)

// listColumns are the fields the list tabular formats render, in the reference
// tool's TSV column order. The names are the index record's field names, so a
// consumer can switch between the index and the command output.
var listColumns = []string{"id", "kind", "status", "updated", "repos", "tags", "codex_sessions", "path", "title"}

// searchColumns are the fields the search tabular formats render, in the
// reference tool's TSV column order.
var searchColumns = []string{"id", "kind", "location", "snippet"}

// checkFormat validates --format and folds --json into it: --json is --format
// jsonl, and an explicit --format that disagrees with it is a usage error rather
// than a silent winner.
func checkFormat(cmd *cobra.Command, jsonOut bool) (string, error) {
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return "", err
	}
	switch format {
	case formatTSV, formatTable, formatCSV, formatJSONL:
	default:
		return "", usagef("invalid argument %q for \"--format\" flag: must be tsv, table, csv, or jsonl", format)
	}
	if jsonOut {
		if cmd.Flags().Changed("format") && format != formatJSONL {
			return "", usagef("--json cannot be combined with --format %s", format)
		}
		format = formatJSONL
	}
	return format, nil
}

// writeList renders the records in one of the four list formats. Every format
// but jsonl shares listCells, so switching formats changes only the framing.
func writeList(cmd *cobra.Command, format string, records []vault.IndexRecord) error {
	out := cmd.OutOrStdout()
	switch format {
	case formatJSONL:
		_, err := out.Write(vault.EncodeIndex(records))
		return err
	case formatCSV:
		return writeCSV(out, listColumns, formatRows(records, listCells))
	case formatTable:
		painter, err := displayStyle(cmd, false)
		if err != nil {
			return err
		}
		return writeTable(out, listColumns, formatRows(records, listCells), painter)
	default:
		return writeTSV(out, formatRows(records, listCells))
	}
}

// writeSearch renders the matches in one of the four search formats.
func writeSearch(cmd *cobra.Command, format string, matches []vault.Match) error {
	out := cmd.OutOrStdout()
	switch format {
	case formatJSONL:
		return writeMatchJSONL(out, matches)
	case formatCSV:
		return writeCSV(out, searchColumns, formatRows(matches, searchCells))
	case formatTable:
		painter, err := displayStyle(cmd, false)
		if err != nil {
			return err
		}
		return writeTable(out, searchColumns, formatRows(matches, searchCells), painter)
	default:
		return writeTSV(out, formatRows(matches, searchCells))
	}
}

// listCells renders one record as the nine cells every list format but JSONL
// shares.
func listCells(record vault.IndexRecord) []string {
	return []string{
		record.ID, record.Kind, orDash(record.Status), record.Updated,
		joinOrDash(record.Repos), joinOrDash(record.Tags), joinOrDash(record.CodexSessions),
		record.Path, record.Title,
	}
}

// searchCells renders one match as the four cells every search format but JSONL
// shares: the reference tool's id, kind, path with its line, and snippet.
func searchCells(match vault.Match) []string {
	location := match.Path
	if match.Line > 0 {
		location = fmt.Sprintf("%s:%d", match.Path, match.Line)
	}
	return []string{match.ID, match.Kind, location, match.Snippet}
}

// formatRows maps items through a cell builder.
func formatRows[T any](items []T, cells func(T) []string) [][]string {
	rows := make([][]string, len(items))
	for i, item := range items {
		rows[i] = cells(item)
	}
	return rows
}

// writeTSV writes rows tab-separated with a newline after every row, the shape
// the reference tool prints.
func writeTSV(out io.Writer, rows [][]string) error {
	for _, row := range rows {
		if _, err := fmt.Fprintln(out, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return nil
}

// writeCSV writes a header row followed by rows as RFC 4180 CSV.
func writeCSV(out io.Writer, header []string, rows [][]string) error {
	w := csv.NewWriter(out)
	if err := w.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// writeTable writes rows in space-aligned columns under a bolded header. Padding
// counts runes, not display width, so a wide (CJK) glyph misaligns its column.
//
// shortcut: rune-count padding; upgrade to a width-aware table if callers start
// parsing the table format.
func writeTable(out io.Writer, header []string, rows [][]string, painter style) error {
	widths := make([]int, len(header))
	for i, name := range header {
		widths[i] = utf8.RuneCountInString(name)
	}
	for _, row := range rows {
		for i, cell := range row {
			if w := utf8.RuneCountInString(cell); i < len(widths) && w > widths[i] {
				widths[i] = w
			}
		}
	}
	writeRow := func(cells []string, colour bool) error {
		parts := make([]string, len(cells))
		for i, cell := range cells {
			parts[i] = cell
			if i < len(cells)-1 { // never pad the last column: no trailing blanks
				parts[i] += strings.Repeat(" ", widths[i]-utf8.RuneCountInString(cell))
			}
		}
		line := strings.Join(parts, "  ")
		if colour {
			line = painter.bold(line)
		}
		_, err := fmt.Fprintln(out, line)
		return err
	}
	if err := writeRow(header, true); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writeRow(row, false); err != nil {
			return err
		}
	}
	return nil
}

// writeMatchJSONL writes one JSON object per search match, carrying the index's
// field names for the fields it has — id, kind and path — plus the matched line
// (0 for a metadata match) and the snippet. A match is not an index record, so
// this uses encoding/json rather than the index encoder; HTML escaping is off so
// a path or snippet keeps its own bytes.
func writeMatchJSONL(out io.Writer, matches []vault.Match) error {
	type matchRecord struct {
		ID      string `json:"id"`
		Kind    string `json:"kind"`
		Path    string `json:"path"`
		Line    int    `json:"line"`
		Snippet string `json:"snippet"`
	}
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	for _, match := range matches {
		if err := enc.Encode(matchRecord{ID: match.ID, Kind: match.Kind, Path: match.Path, Line: match.Line, Snippet: match.Snippet}); err != nil {
			return err
		}
	}
	return nil
}

// writePathJSON reports the resolved root and the selector level that chose it
// (design D8).
func writePathJSON(out io.Writer, resolution vault.Resolution) error {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	return enc.Encode(struct {
		Root   string `json:"root"`
		Source string `json:"source"`
	}{Root: resolution.Root, Source: string(resolution.Source)})
}

// indexRecordFor finds the index record for a resolved document path, so show
// --json emits the same shape as every other structured read. A file outside the
// vault has no index record and is an error.
func indexRecordFor(root, path string) (vault.IndexRecord, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return vault.IndexRecord{}, err
	}
	records, err := vault.Index(root)
	if err != nil {
		return vault.IndexRecord{}, err
	}
	for _, record := range records {
		if filepath.Join(root, filepath.FromSlash(record.Path)) == abs {
			return record, nil
		}
	}
	return vault.IndexRecord{}, fmt.Errorf("%s is not a document in this vault", path)
}
