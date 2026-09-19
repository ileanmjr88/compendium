package pkglock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPreflightOK: every covered file present, regular and readable means no
// error at all. errors.Join returns nil for an empty slice, so the happy path
// needs no special-casing inside preflight.
func TestPreflightOK(t *testing.T) {
	err := preflight("testdata/vcpkg-sibling", "vcpkg",
		[]string{"vcpkg.json", "vcpkg-configuration.json"})
	if err != nil {
		t.Errorf("want nil, got:\n%v", err)
	}
}

func TestPreflightMissingFile(t *testing.T) {
	err := preflight("testdata/vcpkg-alone", "vcpkg", []string{"nope.json"})
	if err == nil {
		t.Fatal("want an error for a missing file, got nil")
	}

	msg := err.Error()
	// The message is user-facing and quoted in ION-20 §4: it has to name the
	// ecosystem, say where the file was expected, and suggest a way out.
	for _, want := range []string{`"vcpkg"`, `"nope.json"`, "not found", "expected at:", "hint:"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message is missing %q:\n%s", want, msg)
		}
	}
}

// TestPreflightDirectory: a directory where a file is expected must stop at
// "not a regular file". Opening a directory succeeds on Unix, so without the
// continue in that branch the same path would also be run through the
// readability check and produce a second, contradictory message.
func TestPreflightDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "vcpkg.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := preflight(root, "vcpkg", []string{"vcpkg.json"})
	if err == nil {
		t.Fatal("want an error for a directory, got nil")
	}
	if !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("want a not-a-regular-file error, got:\n%v", err)
	}
	if strings.Contains(err.Error(), "not readable") {
		t.Errorf("directory reached the readability check — missing continue:\n%v", err)
	}
}

// TestPreflightUnreadable: Stat cannot answer "is this readable" — permission
// bits lie once root, ACLs or mount options are involved — so preflight opens
// the file. This is what proves it does.
func TestPreflightUnreadable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "requirements.txt"), []byte("x\n"), 0o000); err != nil {
		t.Fatal(err)
	}

	err := preflight(root, "pip", []string{"requirements.txt"})
	if err == nil {
		t.Skip("file was readable anyway (running as root?) — permissions not enforced here")
	}
	if !strings.Contains(err.Error(), "not readable") {
		t.Errorf("want a not-readable error, got:\n%v", err)
	}
}

// TestPreflightCollectsAll is the point of the whole function: one run has to
// surface every problem, so the user fixes them in one pass instead of
// rerunning into the next one each time.
func TestPreflightCollectsAll(t *testing.T) {
	err := preflight("testdata/vcpkg-alone", "vcpkg",
		[]string{"a.json", "b.json", "c.json"})
	if err == nil {
		t.Fatal("want an error, got nil")
	}

	if n := strings.Count(err.Error(), "was not found"); n != 3 {
		t.Errorf("reported %d failures, want 3 — preflight is returning early:\n%v", n, err)
	}
}

// A covered file that exists alongside one that does not must not mask the
// failure: preflight reports the bad path and stays quiet about the good one.
func TestPreflightPartialFailure(t *testing.T) {
	err := preflight("testdata/vcpkg-sibling", "vcpkg",
		[]string{"vcpkg.json", "nope.json"})
	if err == nil {
		t.Fatal("want an error, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "nope.json") {
		t.Errorf("missing file not reported:\n%s", msg)
	}
	if strings.Contains(msg, `"vcpkg.json"`) {
		t.Errorf("the healthy file was reported as a failure:\n%s", msg)
	}
}
