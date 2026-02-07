//go:build integration
// +build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildServerBinary builds cmd/mysql-mcp-server into dir and returns the executable path.
func buildServerBinary(t *testing.T, dir string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// Find module root (directory containing go.mod)
	moduleRoot := wd
	for {
		if _, err := os.Stat(filepath.Join(moduleRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(moduleRoot)
		if parent == moduleRoot {
			t.Fatal("could not find go.mod to determine module root")
		}
		moduleRoot = parent
	}
	exe := filepath.Join(dir, "mysql-mcp-server")
	if goexe := os.Getenv("GOEXE"); goexe != "" {
		exe += goexe
	}
	cmd := exec.Command("go", "build", "-o", exe, "./cmd/mysql-mcp-server")
	cmd.Dir = moduleRoot
	if err := cmd.Run(); err != nil {
		t.Skipf("could not build mysql-mcp-server: %v (skip if go not in PATH)", err)
	}
	return exe
}

// TestSilentVersion runs the server binary with --silent --version and verifies
// it exits 0 and prints version info (no INFO/WARN lines when --silent).
func TestSilentVersion(t *testing.T) {
	exe := buildServerBinary(t, t.TempDir())

	run := exec.Command(exe, "--silent", "--version")
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("--silent --version failed: %v\noutput: %s", err, out)
	}
	output := string(out)
	if !strings.Contains(output, "mysql-mcp-server") {
		t.Errorf("expected version output to contain 'mysql-mcp-server'; got: %s", output)
	}
	// With --silent, we should not see [INFO] or [WARN] in version output (version is printf to stdout)
	if strings.Contains(output, "[INFO]") || strings.Contains(output, "[WARN]") {
		t.Errorf("--silent should not produce INFO/WARN; got: %s", output)
	}
}

// TestHelpContainsSilentAndDaemon ensures the CLI help documents --silent and --daemon.
func TestHelpContainsSilentAndDaemon(t *testing.T) {
	exe := buildServerBinary(t, t.TempDir())

	run := exec.Command(exe, "--help")
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\noutput: %s", err, out)
	}
	output := string(out)
	if !strings.Contains(output, "--silent") || !strings.Contains(output, "--daemon") {
		t.Errorf("help should document --silent and --daemon; got (excerpt): %s", output[:min(400, len(output))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
