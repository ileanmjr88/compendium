package main_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var (
	binaryPath string
	covDir     string
)

// TestMain builds the binary once with known ldflags, then runs all tests
// against that binary as a subprocess. Lets us assert real exit codes and
// stdout/stderr without refactoring every cobra Run closure.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "compendium-clitest-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "tempdir:", err)
		os.Exit(1)
	}

	binaryPath = filepath.Join(tmpDir, "compendium")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	covDir = filepath.Join(tmpDir, "covdata")
	if err := os.MkdirAll(covDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "covdir:", err)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	build := exec.Command(
		"go", "build",
		"-cover",
		"-ldflags",
		"-X github.com/ileanmjr88/compendium/internal/buildinfo.Version=test-0.0.0 "+
			"-X github.com/ileanmjr88/compendium/internal/buildinfo.Commit=testcommit",
		"-o", binaryPath,
		".",
	)
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "build:", err)
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()

	// Convert raw covdata emitted by the subprocess into a Go text profile
	// alongside the regular coverage.out at the repo root, so `make coverage`
	// can merge them.
	out := filepath.Join("..", "..", "cli-coverage.out")
	conv := exec.Command("go", "tool", "covdata", "textfmt", "-i="+covDir, "-o="+out)
	conv.Stderr = os.Stderr
	if err := conv.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "covdata textfmt:", err)
	}

	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

func TestVersionCommand(t *testing.T) {
	cmd := exec.Command(binaryPath, "version")
	cmd.Env = append(os.Environ(), "GOCOVERDIR="+covDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compendium version: %v\n%s", err, out)
	}
	got := string(out)
	if !strings.Contains(got, "compendium test-0.0.0") {
		t.Errorf("missing injected version, got:\n%s", got)
	}
	if !strings.Contains(got, "testcommit") {
		t.Errorf("missing injected commit, got:\n%s", got)
	}
	if !strings.Contains(got, runtime.GOOS+"/"+runtime.GOARCH) {
		t.Errorf("missing os/arch line, got:\n%s", got)
	}
}

func TestActivateCommand(t *testing.T) {
	tmpHome := t.TempDir()
	tmpCwd := t.TempDir()

	goBinDir := filepath.Join(tmpHome, ".local", "compendium", "languages", "go", "1.22.0", "bin")
	if err := os.MkdirAll(goBinDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configContent := `[compendium]
version = "test"
min_compendium = "0.1.0"
name = "test-proj"

[registry]
source = "public"

[languages]
go = "1.22.0"
`
	if err := os.WriteFile(filepath.Join(tmpCwd, "compendium.toml"), []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binaryPath, "activate")
	cmd.Dir = tmpCwd
	cmd.Env = []string{
		"HOME=" + tmpHome,
		"PATH=" + os.Getenv("PATH"),
		"GOCOVERDIR=" + covDir,
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("compendium activate: %v\nstderr: %s", err, stderr.String())
	}

	script := stdout.String()
	if !strings.Contains(script, "_COMPENDIUM_ACTIVE") {
		t.Errorf("script missing _COMPENDIUM_ACTIVE guard:\n%s", script)
	}
	if !strings.Contains(script, goBinDir) {
		t.Errorf("script missing go bin path %q:\n%s", goBinDir, script)
	}
}

func TestEnvCommandNoConfig(t *testing.T) {
	cmd := exec.Command(binaryPath, "env")
	cmd.Dir = t.TempDir()
	cmd.Env = []string{
		"HOME=" + t.TempDir(),
		"PATH=" + os.Getenv("PATH"),
		"GOCOVERDIR=" + covDir,
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code: want 1, got %d", exitErr.ExitCode())
	}

	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "loading config") {
		t.Errorf("expected 'loading config' in output, got:\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}
}
