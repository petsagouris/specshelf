package cli

import (
	"runtime/debug"
)

// version is set at build time with -ldflags "-X .../internal/cli.version=...".
// Binaries installed with `go install` never receive ldflags, so the module
// version recorded in the binary is the primary source and this is the override.
var version string

// Version reports the version of this build. It returns the module version
// recorded at build time when there is one, and a development marker otherwise.
func Version() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "dev"
	}
	return info.Main.Version
}
