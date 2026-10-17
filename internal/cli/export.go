package cli

import (
	"github.com/petsagouris/specshelf/internal/vault"
	"github.com/spf13/cobra"
)

// The two export formats. json is a single array; jsonl is the same objects one
// per line, byte-identical to index.jsonl.
const (
	exportJSON  = "json"
	exportJSONL = "jsonl"
)

// newExportCommand writes every document in the vault to stdout, once each, with
// the index record's fields. It is tolerant like list: a document it cannot read
// is a stderr warning, not a failure.
func newExportCommand() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export every document as JSON or JSONL.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolution, err := resolveVault(cmd)
			if err != nil {
				return err
			}
			if format != exportJSON && format != exportJSONL {
				return usagef("invalid argument %q for \"--format\" flag: must be json or jsonl", format)
			}
			records, warnings, err := vault.List(resolution.Root)
			if err != nil {
				return err
			}
			printWarnings(cmd, warnings)

			out := cmd.OutOrStdout()
			if format == exportJSONL {
				_, err = out.Write(vault.EncodeIndex(records))
				return err
			}
			_, err = out.Write(encodeJSONArray(records))
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", exportJSON, "Output format: json or jsonl.")
	return cmd
}

// encodeJSONArray wraps the index encoder's per-record bytes in a JSON array.
// Each element keeps the encoder's spacing, so a jsonl line and the matching
// json element are byte-identical and there is only one record shape.
func encodeJSONArray(records []vault.IndexRecord) []byte {
	out := []byte{'['}
	for i, record := range records {
		if i > 0 {
			out = append(out, ',')
		}
		data, _ := record.MarshalJSON() // MarshalJSON never returns an error
		out = append(out, data...)
	}
	return append(out, ']', '\n')
}
