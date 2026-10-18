package tui

import (
	"fmt"
	"strings"
)

// Binding is one session key: the keys that trigger it and what they do.
// Bindings is the single source of truth shared by the help view and the README,
// which the README test checks against each other so they cannot drift.
type Binding struct {
	Keys  []string
	Label string
}

// Bindings lists the session's keys in the order the help view and the README
// present them.
var Bindings = []Binding{
	{[]string{"j", "down"}, "move the selection down"},
	{[]string{"k", "up"}, "move the selection up"},
	{[]string{"enter"}, "open the selected document"},
	{[]string{"esc"}, "go back or cancel"},
	{[]string{"e"}, "edit the open document in your editor"},
	{[]string{"s"}, "set the open document's status"},
	{[]string{"d"}, "delete the open document (confirm with y)"},
	{[]string{"/"}, "filter the list as you type"},
	{[]string{"K"}, "cycle the kind filter"},
	{[]string{"S"}, "cycle the status filter"},
	{[]string{"R"}, "cycle the repo filter"},
	{[]string{"T"}, "cycle the tag filter"},
	{[]string{"r"}, "reload the vault from disk"},
	{[]string{"c"}, "create a document"},
	{[]string{"tab"}, "switch kind in the create prompt"},
	{[]string{"?"}, "show this help"},
	{[]string{"q", "ctrl+c"}, "quit"},
}

// keyLine renders one binding: the keys, two spaces, and the label. The two
// spaces are what the README test splits on when it reads the rendered help.
func (b Binding) keyLine() string {
	return fmt.Sprintf("%-14s  %s", strings.Join(b.Keys, ", "), b.Label)
}
