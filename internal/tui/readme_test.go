package tui

import (
	"os"
	"strings"
	"testing"
)

// TestReadmeDocumentsSessionKeys asserts the keys the README documents are
// exactly the keys the help view renders, in the same order, so the two cannot
// drift.
func TestReadmeDocumentsSessionKeys(t *testing.T) {
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	documented := readmeKeys(string(data))
	if len(documented) == 0 {
		t.Fatal("no session keys found in README.md under \"Interactive session\"")
	}
	rendered := helpKeys(t)
	if len(documented) != len(rendered) {
		t.Fatalf("README documents %d key rows, the help view renders %d\ndocumented: %v\nrendered:   %v",
			len(documented), len(rendered), documented, rendered)
	}
	for i := range documented {
		if strings.Join(documented[i], ",") != strings.Join(rendered[i], ",") {
			t.Fatalf("key row %d: README has %v, the help view renders %v", i, documented[i], rendered[i])
		}
	}
}

// readmeKeys parses the markdown table under the "Interactive session" heading
// and returns each row's keys, split on commas.
func readmeKeys(readme string) [][]string {
	var rows [][]string
	inSection := false
	for _, line := range strings.Split(readme, "\n") {
		if strings.HasPrefix(line, "## ") {
			inSection = strings.TrimSpace(line) == "## Interactive session"
			continue
		}
		if !inSection || !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 2 {
			continue
		}
		keys := splitKeys(cells[1])
		if len(keys) == 0 || keys[0] == "Key" || strings.Trim(keys[0], "-") == "" {
			continue
		}
		rows = append(rows, keys)
	}
	return rows
}

// helpKeys renders the help view and returns each binding line's keys.
func helpKeys(t *testing.T) [][]string {
	t.Helper()
	m, _ := newFake()
	update(m, k('?'))
	var rows [][]string
	for _, line := range strings.Split(m.View().Content, "\n") {
		index := strings.Index(line, "  ")
		if index < 0 {
			continue
		}
		if keys := splitKeys(line[:index]); len(keys) > 0 {
			rows = append(rows, keys)
		}
	}
	return rows
}

// splitKeys splits a key cell into its keys, dropping backticks and whitespace.
func splitKeys(cell string) []string {
	var keys []string
	for _, part := range strings.Split(cell, ",") {
		key := strings.TrimSpace(strings.ReplaceAll(part, "`", ""))
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}
