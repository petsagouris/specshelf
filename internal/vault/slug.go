package vault

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// slugify derives the id segment from a title, matching the Python reference
// implementation: NFKD normalise, keep ASCII, lowercase, collapse anything that
// is not [a-z0-9] to a single "-", trim, cap at 72 bytes and trim again.
// Titles that survive as nothing fall back to the first 10 hex of sha1(title).
func slugify(title string) string {
	var ascii strings.Builder
	for _, r := range norm.NFKD.String(title) {
		if r <= unicode.MaxASCII {
			ascii.WriteRune(r)
		}
	}

	words := strings.FieldsFunc(strings.ToLower(ascii.String()), func(r rune) bool {
		return !isAlnum(r)
	})
	slug := strings.Join(words, "-")

	// The string is pure ASCII here, so byte slicing matches Python's character
	// slicing.
	if len(slug) > 72 {
		slug = strings.Trim(slug[:72], "-")
	}
	if slug == "" {
		sum := sha1.Sum([]byte(title))
		return hex.EncodeToString(sum[:])[:10]
	}
	return slug
}

// isAlnum reports whether r is an ASCII lowercase letter or digit, the only
// characters the reference implementation keeps in a slug.
func isAlnum(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
}
