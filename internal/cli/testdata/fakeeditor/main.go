// Command fakeeditor is the stand-in editor the command scripts put in $VISUAL.
// It appends the value of FAKE_EDITOR_APPEND to the document named by its last
// argument and exits 0, so an unset variable is an editor that changed nothing.
package main

import "os"

func main() {
	if len(os.Args) < 2 {
		os.Exit(2)
	}
	suffix := os.Getenv("FAKE_EDITOR_APPEND")
	if suffix == "" {
		return
	}
	f, err := os.OpenFile(os.Args[len(os.Args)-1], os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		os.Exit(1)
	}
	if _, err := f.WriteString(suffix); err != nil {
		_ = f.Close()
		os.Exit(1)
	}
	_ = f.Close()
}
