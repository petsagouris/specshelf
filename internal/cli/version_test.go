package cli

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestVersionFromBuildInfo(t *testing.T) {
	tests := []struct {
		name string
		main string
		vcs  bool
		want string
	}{
		{name: "no version recorded", main: "", want: "dev"},
		{name: "devel placeholder", main: "(devel)", want: "dev"},
		{name: "installed module version", main: "v0.1.0", want: "v0.1.0"},
		{
			name: "pseudo-version from a checkout",
			main: "v0.0.0-20261017211210-b8e7d89971e3",
			vcs:  true,
			want: "dev (v0.0.0-20261017211210-b8e7d89971e3)",
		},
		{name: "tagged version with vcs stamping keeps the tag visible", main: "v0.1.0", vcs: true, want: "dev (v0.1.0)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &debug.BuildInfo{}
			info.Main.Version = tt.main
			if tt.vcs {
				info.Settings = append(info.Settings, debug.BuildSetting{Key: "vcs.revision", Value: "b8e7d89971e3"})
			}
			if got := versionFrom(info, true); got != tt.want {
				t.Errorf("versionFrom(main=%q, vcs=%v) = %q, want %q", tt.main, tt.vcs, got, tt.want)
			}
		})
	}

	t.Run("no build information", func(t *testing.T) {
		if got := versionFrom(nil, false); got != "dev" {
			t.Errorf("versionFrom(nil, false) = %q, want dev", got)
		}
	})
}

// A binary built from a checkout reports a development marker even though it
// carries a usable pseudo-version, which is what the install smoke test asserts.
func TestVersionOverrideWins(t *testing.T) {
	previous := version
	version = "v9.9.9-injected"
	t.Cleanup(func() { version = previous })

	if got := Version(); got != "v9.9.9-injected" {
		t.Errorf("Version() = %q, want the injected value", got)
	}
	if !strings.Contains(Version(), "v9.9.9") {
		t.Error("the injected version is not the one reported")
	}
}
