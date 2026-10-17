// Package config loads user preferences from a TOML file and resolves them
// against command-line flags and environment variables.
//
// # File location
//
// Path reports the file the tool reads by default: the file named by the
// SPECSHELF_CONFIG environment variable when it is set and non-empty, and
// otherwise config.toml inside a specshelf directory under the platform's
// conventional user configuration directory (os.UserConfigDir). Load reads the
// file it is given; a file that does not exist is not an error and leaves every
// setting at its built-in default.
//
// # Precedence
//
// Resolve applies the precedence for one setting — flag, then environment
// variable, then configuration file, then built-in default — and reports which
// level supplied the value. A nil pointer means that level did not set the
// setting. The package never reads flags or per-setting environment variables:
// the caller passes those levels in, so every surface resolves identically and
// tests can exercise each level in isolation.
//
// # The file
//
// File is the decoded configuration file. Every scalar field is a pointer so a
// caller can tell an unset setting from one set to the zero value, and named
// vaults are keyed by their declared name. The recognised keys are documented in
// docs/CONFIG.md, whose sample block a test parses with Parse:
//
//	default_vault = "~/.agents/spec-vault"
//	color         = "auto"
//	editor        = "vim"
//	auto_index    = true
//	project       = true
//	[git]
//	remote = "origin"
//	push   = true
//	[vaults.work]
//	path = "~/vaults/work"
//
// # Errors and warnings
//
// Parse returns an error naming the file and the line and column of a malformed
// or wrongly-typed value; it never falls back to defaults silently. A key the
// tool does not know is returned as a warning instead, so the caller can print
// it and carry on.
package config
