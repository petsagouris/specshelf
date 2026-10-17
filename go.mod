module github.com/petsagouris/specshelf

go 1.26.0

require (
	github.com/spf13/cobra v1.10.2
	go.yaml.in/yaml/v3 v3.0.5 // test-only: asserts the frontmatter we emit is valid YAML
	golang.org/x/text v0.43.0
)

require (
	github.com/pelletier/go-toml/v2 v2.4.3
	github.com/rogpeppe/go-internal v1.16.0 // test-only: runs the per-command testscript files
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/tools v0.51.0 // indirect
)
