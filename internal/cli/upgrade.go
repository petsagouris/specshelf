package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// modulePath is the module this binary is installed from; commandPath is the
// package `go install` builds; installCommand is what a user runs to update.
const (
	modulePath     = "github.com/petsagouris/specshelf"
	commandPath    = modulePath + "/cmd/specshelf"
	installCommand = "go install " + commandPath + "@latest"
)

// proxyBaseURL is the Go module proxy `upgrade --check` queries. It is a
// variable so tests can point it at a local server; outside tests it is the
// only network endpoint the program uses (design D10).
var proxyBaseURL = "https://proxy.golang.org"

// upgradeGOOS is runtime.GOOS, a variable so the Windows no-replace path is
// testable on any platform.
var upgradeGOOS = runtime.GOOS

// proxyTimeout bounds the one request --check makes, so an unreachable proxy
// fails the command rather than hanging it.
const proxyTimeout = 15 * time.Second

// newUpgradeCommand checks for a newer version and updates the binary when it
// is safe to do so. --check only reports; upgrade installs only when the running
// executable is in the directory `go install` writes to, and otherwise prints
// the command to run (design D10).
func newUpgradeCommand() *cobra.Command {
	var check bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Check for and install a newer version.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if check {
				return runUpgradeCheck(cmd.OutOrStdout())
			}
			return runUpgrade(cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Report the running and latest versions without changing anything.")
	return cmd
}

// runUpgradeCheck reports the running and latest versions. It exits
// successfully whether or not an update is available and never writes anything;
// a proxy that cannot be reached is the error. See design D10.
func runUpgradeCheck(out io.Writer) error {
	latest, err := latestVersion()
	if err != nil {
		return err
	}
	running := Version()
	if _, err := fmt.Fprintf(out, "running %s\nlatest %s\n", running, latest); err != nil {
		return err
	}
	verdict := "an update is available"
	if running == latest {
		verdict = "already up to date"
	}
	_, err = fmt.Fprintln(out, verdict)
	return err
}

// latestVersion queries the module proxy for the module's latest version.
func latestVersion() (string, error) {
	url := proxyBaseURL + "/" + modulePath + "/@latest"
	client := &http.Client{Timeout: proxyTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("query the module proxy: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("module proxy returned %s for %s", resp.Status, url)
	}
	var payload struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode the module proxy response: %w", err)
	}
	if payload.Version == "" {
		return "", fmt.Errorf("module proxy returned no version for %s", url)
	}
	return payload.Version, nil
}

// runUpgrade installs the latest version when the running executable lives in
// the directory `go install` writes to. On Windows, and for a binary copied
// anywhere else, it prints the install command and the running executable's
// path and succeeds without changing anything. When it would install but the Go
// toolchain is missing, it reports that instead. See design D10.
func runUpgrade(out io.Writer) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate the running executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	target := installDir()
	if upgradeGOOS == "windows" {
		return printInstallCommand(out, "the running executable cannot be replaced on Windows; run this in a fresh shell:")
	}
	if target == "" || !sameDir(filepath.Dir(exe), target) {
		return printInstallCommand(out, fmt.Sprintf("%s is not in the go install directory; run:", exe))
	}

	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("the Go toolchain is required to upgrade %s; run: %s", exe, installCommand)
	}
	output, err := exec.Command("go", "install", commandPath+"@latest").CombinedOutput()
	if err != nil {
		return fmt.Errorf("go install %s@latest: %s", commandPath, strings.TrimSpace(string(output)))
	}
	if version, err := installedVersion(filepath.Join(target, "specshelf")); err == nil {
		_, err = fmt.Fprintf(out, "installed %s\n", version)
		return err
	}
	_, err = fmt.Fprintf(out, "installed %s@latest\n", commandPath)
	return err
}

// printInstallCommand writes the instructions for a binary the tool will not
// replace: a reason line naming the running executable, then the exact command.
func printInstallCommand(out io.Writer, reason string) error {
	if _, err := fmt.Fprintln(out, reason); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, installCommand)
	return err
}

// installedVersion asks the freshly installed binary to report its version, so
// upgrade names the version it installed rather than only the command.
func installedVersion(path string) (string, error) {
	output, err := exec.Command(path, "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// installDir is the directory a plain `go install` writes to: GOBIN when set,
// otherwise the first GOPATH's bin, otherwise ~/go/bin.
func installDir() string {
	if dir := os.Getenv("GOBIN"); dir != "" {
		return dir
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return filepath.Join(filepath.SplitList(gopath)[0], "bin")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "go", "bin")
}

// sameDir compares two directories by their resolved, cleaned form.
func sameDir(a, b string) bool {
	if resolved, err := filepath.EvalSymlinks(a); err == nil {
		a = resolved
	}
	if resolved, err := filepath.EvalSymlinks(b); err == nil {
		b = resolved
	}
	return filepath.Clean(a) == filepath.Clean(b)
}
