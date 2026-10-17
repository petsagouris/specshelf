package vault

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// localDate formats a time as the vault's local date, YYYY-MM-DD.
func localDate(t time.Time) string {
	return t.Format(time.DateOnly)
}

// compactDate turns YYYY-MM-DD into YYYYMMDD.
func compactDate(date string) string {
	return strings.ReplaceAll(date, "-", "")
}

// documentID returns "spec-20261017-a-slug" or "decision-20261017-a-slug".
func documentID(kind, date, slug string) string {
	return kind + "-" + compactDate(date) + "-" + slug
}

// uniqueID appends "-2", "-3", ... to base until taken reports false.
func uniqueID(base string, taken func(string) bool) string {
	if !taken(base) {
		return base
	}
	for n := 2; ; n++ {
		candidate := base + "-" + strconv.Itoa(n)
		if !taken(candidate) {
			return candidate
		}
	}
}

// documentPath returns <dir>/YYYY-MM-DD-<slug>.md.
func documentPath(dir, date, slug string) string {
	return filepath.Join(dir, date+"-"+slug+".md")
}

// nextFreePath returns path, or path with "-2", "-3", ... before its extension,
// whichever does not exist on disk first. It never creates or modifies a file.
func nextFreePath(path string) string {
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	return uniqueID(stem, func(p string) bool {
		_, err := os.Stat(p + ext)
		return err == nil
	}) + ext
}
