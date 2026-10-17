package cli

import (
	"runtime/debug"
)

// version is set at build time with -ldflags "-X .../internal/cli.version=...".
// Binaries installed with `go install` never receive ldflags, so the module
// version recorded in the binary is the primary source and this is the override.
var version string

// Version reports the version of this build.
func Version() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	return versionFrom(info, ok)
}

// versionFrom turns build information into a reported version.
//
// `go install ...@v1.2.3` records the module version, and so does a build from a
// downloaded module. A build from a version control checkout carries vcs
// stamping and a pseudo-version such as v0.0.0-20261017211210-b8e7d89971e3,
// which is still a development build, so it is reported as one while keeping
// the pseudo-version visible for traceability.
func versionFrom(info *debug.BuildInfo, ok bool) string {
	if !ok || info == nil || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "dev"
	}
	if builtFromWorkingCopy(info) {
		return "dev (" + info.Main.Version + ")"
	}
	return info.Main.Version
}

// builtFromWorkingCopy reports whether the binary carries version control
// stamping, which only happens when it was built from a checkout.
func builtFromWorkingCopy(info *debug.BuildInfo) bool {
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return true
		}
	}
	return false
}
