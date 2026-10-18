package vault

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// configFile points SPECSHELF_CONFIG at a fresh file holding contents. An empty
// contents leaves the file absent, which the loader treats as an empty config.
func configFile(t *testing.T, contents string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if contents != "" {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
	}
	t.Setenv("SPECSHELF_CONFIG", path)
}

const namedVaultsConfig = `default_vault = "/from-config"
[vaults.work]
path = "/named-work"
[vaults.archive]
path = "/named-archive"
`

// assertResolved fails unless the resolution is exactly root from source.
func assertResolved(t *testing.T, res Resolution, err error, root string, source Source) {
	t.Helper()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.Root != root || res.Source != source {
		t.Errorf("Resolve = (%q, %q), want (%q, %q)", res.Root, res.Source, root, source)
	}
}

// nativeRoot is the root Resolve reports for a configured path literal: the
// literal run through filepath.Abs, which is what Resolve itself does to every
// selector. On Unix the literal is already absolute, but a Windows
// rooted-without-volume path like "/from-env" resolves against the current
// drive, so the expectation has to be built natively rather than written with
// "/".
func nativeRoot(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("Abs(%q): %v", path, err)
	}
	return abs
}

func TestResolvePrecedence(t *testing.T) {
	home := t.TempDir()

	t.Run("built-in default when nothing selects a vault", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "")
		configFile(t, "")
		res, err := Resolve(Selectors{}, home)
		assertResolved(t, res, err, filepath.Join(home, ".agents", "spec-vault"), SourceDefault)
		if len(res.Warnings) != 0 {
			t.Errorf("Warnings = %v, want none", res.Warnings)
		}
	})

	t.Run("config default beats the built-in default", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "")
		configFile(t, namedVaultsConfig)
		res, err := Resolve(Selectors{}, home)
		assertResolved(t, res, err, nativeRoot(t, "/from-config"), SourceConfig)
	})

	t.Run("SPECSHELF_HOME beats the config default", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "/from-env")
		configFile(t, namedVaultsConfig)
		res, err := Resolve(Selectors{}, home)
		assertResolved(t, res, err, nativeRoot(t, "/from-env"), SourceEnv)
	})

	t.Run("SPECSHELF_HOME beats --vault and the config default", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "/from-env")
		configFile(t, namedVaultsConfig)
		res, err := Resolve(Selectors{Vault: "work"}, home)
		assertResolved(t, res, err, nativeRoot(t, "/from-env"), SourceEnv)
	})

	t.Run("--home beats SPECSHELF_HOME and the config default", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "/from-env")
		configFile(t, namedVaultsConfig)
		res, err := Resolve(Selectors{Home: "/from-flag"}, home)
		assertResolved(t, res, err, nativeRoot(t, "/from-flag"), SourceHome)
	})

	t.Run("a relative --home resolves to an absolute root", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "")
		configFile(t, "")
		dir := t.TempDir()
		t.Chdir(dir)
		res, err := Resolve(Selectors{Home: "relative/vault"}, home)
		assertResolved(t, res, err, filepath.Join(dir, "relative", "vault"), SourceHome)
	})

	t.Run("--vault resolves through the config and beats its default", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "")
		configFile(t, namedVaultsConfig)
		res, err := Resolve(Selectors{Vault: "work"}, home)
		assertResolved(t, res, err, nativeRoot(t, "/named-work"), SourceVault)
	})

	t.Run("--project beats the config default", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "")
		configFile(t, namedVaultsConfig)
		project := filepath.Join(home, "work", "project")
		marker := filepath.Join(project, ".specshelf")
		if err := os.MkdirAll(marker, 0o755); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(project, "internal", "vault")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		res, err := Resolve(Selectors{Project: true, Dir: dir}, home)
		assertResolved(t, res, err, marker, SourceProject)
	})

	t.Run("~ in the config default is expanded against home", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "")
		configFile(t, "default_vault = \"~/cfg-vault\"\n")
		res, err := Resolve(Selectors{}, home)
		assertResolved(t, res, err, filepath.Join(home, "cfg-vault"), SourceConfig)
	})

	t.Run("~ in a named vault path is expanded against home", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "")
		configFile(t, "[vaults.home]\npath = \"~/named-vault\"\n")
		res, err := Resolve(Selectors{Vault: "home"}, home)
		assertResolved(t, res, err, filepath.Join(home, "named-vault"), SourceVault)
	})

	t.Run("unknown keys are reported as warnings", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "")
		configFile(t, "colour = \"red\"\n")
		res, err := Resolve(Selectors{}, home)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "colour") {
			t.Errorf("Warnings = %v, want one naming %q", res.Warnings, "colour")
		}
	})
}

func TestResolveConflicts(t *testing.T) {
	home := t.TempDir()
	cases := []struct {
		name          string
		sel           Selectors
		first, second string
	}{
		{"home and vault", Selectors{Home: "/a", Vault: "work"}, "--home", "--vault"},
		{"home and project", Selectors{Home: "/a", Project: true}, "--home", "--project"},
		{"vault and project", Selectors{Vault: "work", Project: true}, "--vault", "--project"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("SPECSHELF_HOME", "")
			configFile(t, namedVaultsConfig)
			res, err := Resolve(c.sel, home)
			var conflict *ConflictError
			if !errors.As(err, &conflict) {
				t.Fatalf("Resolve error = %v, want *ConflictError", err)
			}
			if res.Root != "" || res.Source != "" {
				t.Errorf("Resolve = %+v, want the zero resolution", res)
			}
			msg := err.Error()
			if !strings.Contains(msg, c.first) || !strings.Contains(msg, c.second) {
				t.Errorf("error %q must name both %s and %s", msg, c.first, c.second)
			}
		})
	}
}

// TestResolveConflictPrecedesConfig proves the conflict is rejected before the
// configuration file is read: the config here is malformed, so a config read
// would fail differently.
func TestResolveConflictPrecedesConfig(t *testing.T) {
	t.Setenv("SPECSHELF_HOME", "")
	configFile(t, "default_vault = [unterminated\n")
	_, err := Resolve(Selectors{Home: "/a", Vault: "work"}, t.TempDir())
	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("Resolve error = %v, want *ConflictError", err)
	}
}

// The config project setting is a kill switch for the --project level: unset
// means enabled as before, false refuses an explicit --project with a typed
// error naming the setting instead of resolving another vault.
func TestResolveProjectSetting(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(home, "work", "project")
	marker := filepath.Join(project, ".specshelf")
	if err := os.MkdirAll(marker, 0o755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(project, "internal", "vault")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("unset leaves project discovery enabled", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "")
		configFile(t, "default_vault = \"/from-config\"\n")
		res, err := Resolve(Selectors{Project: true, Dir: dir}, home)
		assertResolved(t, res, err, marker, SourceProject)
	})

	t.Run("false refuses an explicit --project", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "")
		configFile(t, "project = false\ndefault_vault = \"/from-config\"\n")
		res, err := Resolve(Selectors{Project: true, Dir: dir}, home)
		var disabled *ProjectDisabledError
		if !errors.As(err, &disabled) {
			t.Fatalf("Resolve error = %v, want *ProjectDisabledError", err)
		}
		if res.Root != "" || res.Source != "" {
			t.Errorf("Resolve = %+v, want the zero resolution and no fallback", res)
		}
		if !strings.Contains(err.Error(), "project") {
			t.Errorf("error %q must name the project setting", err.Error())
		}
	})

	t.Run("false leaves the other selectors alone", func(t *testing.T) {
		t.Setenv("SPECSHELF_HOME", "")
		configFile(t, "project = false\ndefault_vault = \"/from-config\"\n")
		res, err := Resolve(Selectors{}, home)
		assertResolved(t, res, err, nativeRoot(t, "/from-config"), SourceConfig)
	})
}

func TestResolveUnknownVault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPECSHELF_HOME", "")

	t.Run("lists the declared names", func(t *testing.T) {
		configFile(t, namedVaultsConfig)
		_, err := Resolve(Selectors{Vault: "personal"}, home)
		var unknown *UnknownVaultError
		if !errors.As(err, &unknown) {
			t.Fatalf("Resolve error = %v, want *UnknownVaultError", err)
		}
		if unknown.Name != "personal" {
			t.Errorf("Name = %q, want %q", unknown.Name, "personal")
		}
		want := []string{"archive", "work"}
		if strings.Join(unknown.Declared, ",") != strings.Join(want, ",") {
			t.Errorf("Declared = %v, want %v", unknown.Declared, want)
		}
		for _, name := range append([]string{"personal"}, want...) {
			if !strings.Contains(err.Error(), name) {
				t.Errorf("error %q must name %q", err.Error(), name)
			}
		}
	})

	t.Run("no declared vaults", func(t *testing.T) {
		configFile(t, "")
		_, err := Resolve(Selectors{Vault: "personal"}, home)
		var unknown *UnknownVaultError
		if !errors.As(err, &unknown) {
			t.Fatalf("Resolve error = %v, want *UnknownVaultError", err)
		}
		if !strings.Contains(err.Error(), "no vaults are declared") {
			t.Errorf("error = %q, want it to say none are declared", err.Error())
		}
	})
}
