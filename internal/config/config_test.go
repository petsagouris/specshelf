package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/petsagouris/specshelf/internal/config"
)

// Scenario: No configuration file — a missing file is not an error and behaves
// exactly like an empty one.
func TestNoConfigurationFile(t *testing.T) {
	file, warnings, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none", warnings)
	}
	if file.DefaultVault != nil || file.Color != nil || file.Editor != nil ||
		file.AutoIndex != nil || file.Project != nil || file.Git != nil || len(file.Vaults) != 0 {
		t.Errorf("File = %+v, want the empty file", file)
	}
	if got, source := config.Resolve[string](nil, nil, file.Color, "auto"); got != "auto" || source != config.SourceDefault {
		t.Errorf("Resolve = (%q, %s), want (auto, default)", got, source)
	}
}

// Scenario: Override through the environment — SPECSHELF_CONFIG names the file
// and the default location is not read.
func TestOverrideThroughTheEnvironment(t *testing.T) {
	override := filepath.Join(t.TempDir(), "elsewhere.toml")
	if err := os.WriteFile(override, []byte("color = \"never\"\n"), 0o644); err != nil {
		t.Fatalf("write override: %v", err)
	}
	t.Setenv("SPECSHELF_CONFIG", override)

	path, err := config.Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if path != override {
		t.Fatalf("Path() = %q, want %q", path, override)
	}
	file, warnings, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings = %v, want none", warnings)
	}
	if file.Color == nil || *file.Color != "never" {
		t.Errorf("Color = %v, want never", file.Color)
	}
}

// Scenario: Flag beats config — and each level below it in turn.
func TestFlagBeatsConfig(t *testing.T) {
	flag, env, file, def := "flag", "env", "file", "default"
	tests := []struct {
		name     string
		flag     *string
		env      *string
		file     *string
		want     string
		wantFrom config.Source
	}{
		{"flag wins", &flag, &env, &file, "flag", config.SourceFlag},
		{"environment beats config", nil, &env, &file, "env", config.SourceEnv},
		{"config beats default", nil, nil, &file, "file", config.SourceConfig},
		{"default when nothing is set", nil, nil, nil, "default", config.SourceDefault},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, from := config.Resolve(tt.flag, tt.env, tt.file, def)
			if got != tt.want || from != tt.wantFrom {
				t.Errorf("Resolve = (%q, %s), want (%q, %s)", got, from, tt.want, tt.wantFrom)
			}
		})
	}
}

// Scenario: Config applies when nothing overrides it.
func TestConfigAppliesWhenNothingOverridesIt(t *testing.T) {
	file, warnings, err := config.Parse("config.toml", []byte("color = \"always\"\n"))
	if err != nil || len(warnings) != 0 {
		t.Fatalf("Parse = (%v, %v), want no error or warnings", warnings, err)
	}
	if value, source := config.Resolve[string](nil, nil, file.Color, "auto"); value != "always" || source != config.SourceConfig {
		t.Errorf("Resolve = (%q, %s), want (always, config)", value, source)
	}
}

// Requirement: Configurable settings — every documented key decodes into File.
func TestConfigurableSettings(t *testing.T) {
	const data = `
default_vault = "/vaults/default"
color = "never"
editor = "nano"
auto_index = false
project = false

[git]
remote = "upstream"
push = false

[vaults.work]
path = "/vaults/work"
`
	file, warnings, err := config.Parse("config.toml", []byte(data))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	checks := []struct {
		name string
		ok   bool
	}{
		{"default_vault", file.DefaultVault != nil && *file.DefaultVault == "/vaults/default"},
		{"color", file.Color != nil && *file.Color == "never"},
		{"editor", file.Editor != nil && *file.Editor == "nano"},
		{"auto_index", file.AutoIndex != nil && !*file.AutoIndex},
		{"project", file.Project != nil && !*file.Project},
		{"git.remote", file.Git != nil && file.Git.Remote != nil && *file.Git.Remote == "upstream"},
		{"git.push", file.Git != nil && file.Git.Push != nil && !*file.Git.Push},
		{"vaults.work.path", file.Vaults["work"].Path == "/vaults/work"},
	}
	for _, check := range checks {
		if !check.ok {
			t.Errorf("%s did not decode as expected", check.name)
		}
	}
}

// Scenario: Disabling automatic index refresh — the setting comes from the file.
func TestDisablingAutomaticIndexRefresh(t *testing.T) {
	file, _, err := config.Parse("config.toml", []byte("auto_index = false\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, source := config.Resolve[bool](nil, nil, file.AutoIndex, true); got || source != config.SourceConfig {
		t.Errorf("Resolve = (%v, %s), want (false, config)", got, source)
	}
	file, _, err = config.Parse("config.toml", nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, source := config.Resolve[bool](nil, nil, file.AutoIndex, true); !got || source != config.SourceDefault {
		t.Errorf("Resolve = (%v, %s), want (true, default)", got, source)
	}
}

// Scenario: Unknown key — reported as a warning naming it, and the rest of the
// file still applies.
func TestUnknownKey(t *testing.T) {
	file, warnings, err := config.Parse("config.toml", []byte("colour = \"always\"\ncolor = \"never\"\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "colour") {
		t.Fatalf("warnings = %v, want one naming colour", warnings)
	}
	if file.Color == nil || *file.Color != "never" {
		t.Errorf("Color = %v, want never (known keys still apply)", file.Color)
	}
}

// Requirement: Named vaults — [vaults.<name>] sections carry a path.
func TestNamedVaults(t *testing.T) {
	const data = `
[vaults.work]
path = "/vaults/work"

[vaults.personal]
path = "/vaults/personal"
`
	file, warnings, err := config.Parse("config.toml", []byte(data))
	if err != nil || len(warnings) != 0 {
		t.Fatalf("Parse = (%v, %v), want no error or warnings", warnings, err)
	}
	if file.Vaults["work"].Path != "/vaults/work" {
		t.Errorf("vaults.work = %q, want /vaults/work", file.Vaults["work"].Path)
	}
	if file.Vaults["personal"].Path != "/vaults/personal" {
		t.Errorf("vaults.personal = %q, want /vaults/personal", file.Vaults["personal"].Path)
	}
}

// Scenario: Syntax error in the config file — the error names the file and the
// offending line, and nothing falls back to defaults.
func TestMalformedConfigurationIsReported(t *testing.T) {
	t.Run("syntax error", func(t *testing.T) {
		_, _, err := config.Parse("config.toml", []byte("color = \"always\"\n[vaults.work\npath = \"/v\"\n"))
		if err == nil {
			t.Fatal("Parse succeeded, want an error")
		}
		t.Logf("syntax error: %v", err)
		if !strings.Contains(err.Error(), "config.toml:2:") {
			t.Errorf("error = %q, want it to name config.toml line 2", err)
		}
	})

	t.Run("wrongly typed value", func(t *testing.T) {
		_, _, err := config.Parse("config.toml", []byte("auto_index = \"yes\"\n"))
		if err == nil {
			t.Fatal("Parse succeeded, want an error")
		}
		t.Logf("type error: %v", err)
		if !strings.Contains(err.Error(), "config.toml:1:") {
			t.Errorf("error = %q, want it to name config.toml line 1", err)
		}
	})
}

// The sample block in docs/CONFIG.md is the loader's test input, not a copy of
// it: the document is read and its first fenced TOML block parsed.
func TestSampleFromDocsParses(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "CONFIG.md"))
	if err != nil {
		t.Fatalf("read docs/CONFIG.md: %v", err)
	}
	sample := fencedTOML(t, string(doc))

	file, warnings, err := config.Parse("docs/CONFIG.md", sample)
	if err != nil {
		t.Fatalf("Parse sample: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if file.DefaultVault == nil || file.Color == nil || file.Editor == nil ||
		file.AutoIndex == nil || file.Project == nil ||
		file.Git == nil || file.Git.Remote == nil || file.Git.Push == nil {
		t.Fatalf("sample did not set every scalar setting: %+v", file)
	}
	if file.Vaults["work"].Path == "" {
		t.Errorf("sample did not declare the work vault: %+v", file.Vaults)
	}
}

// fencedTOML returns the body of the first ```toml block in doc.
func fencedTOML(t *testing.T, doc string) []byte {
	t.Helper()
	const open, close = "```toml", "```"
	start := strings.Index(doc, open)
	if start < 0 {
		t.Fatal("docs/CONFIG.md has no ```toml block")
	}
	start += len(open)
	if nl := strings.IndexByte(doc[start:], '\n'); nl >= 0 {
		start += nl + 1
	}
	end := strings.Index(doc[start:], close)
	if end < 0 {
		t.Fatal("docs/CONFIG.md has an unterminated ```toml block")
	}
	return []byte(doc[start : start+end])
}
