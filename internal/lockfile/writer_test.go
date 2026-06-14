package lockfile

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const writeHeader = "# compendium.lock\n# AUTO-GENERATED. Do not edit by hand. Commit to version control.\n"

// unsortedLockfile returns a valid Lockfile whose every slice is deliberately
// out of canonical order: languages/tools not sorted by name, artifacts not
// sorted by (platform, arch), packages not sorted by ecosystem. Write must
// reorder all of these into a stable canonical form before serializing.
func unsortedLockfile() Lockfile {
	return Lockfile{
		Meta: Meta{
			Schema:            CurrentSchema,
			Project:           "wordNebula",
			CompendiumVersion: "0.1.1",
			Platforms:         []string{"linux-amd64", "darwin-arm64"},
			Registry:          RegistryMeta{Source: "public", IndexVersion: "2"},
		},
		Languages: []Entry{
			{
				Name: "python", Spec: "3.11.15", Version: "3.11.15", Source: SourceRegistry,
				Artifacts: []Artifact{
					{Platform: "linux", Arch: "amd64", URL: "https://example.com/py-linux.tar.gz", Checksum: "sha256:p1", Size: 1, Strip: 1},
					{Platform: "darwin", Arch: "arm64", URL: "https://example.com/py-darwin.tar.gz", Checksum: "sha256:p2", Size: 2, Strip: 1},
				},
			},
			{
				Name: "go", Spec: "1.24.1", Version: "1.24.1", Source: SourceRegistry,
				Artifacts: []Artifact{
					{Platform: "linux", Arch: "amd64", URL: "https://example.com/go-linux.tar.gz", Checksum: "sha256:g1", Size: 3, Strip: 1},
					{Platform: "darwin", Arch: "arm64", URL: "https://example.com/go-darwin.tar.gz", Checksum: "sha256:g2", Size: 4, Strip: 1},
				},
			},
		},
		Tools: []Entry{
			{
				Name: "ninja", Spec: "1.13.2", Version: "1.13.2", Source: SourceRegistry,
				Artifacts: []Artifact{
					{Platform: "linux", Arch: "amd64", URL: "https://example.com/ninja-linux.zip", Checksum: "sha256:n1", Size: 5},
					{Platform: "darwin", Arch: "arm64", URL: "https://example.com/ninja-mac.zip", Checksum: "sha256:n2", Size: 6},
				},
			},
			{
				Name: "cmake", Spec: "4.3.1", Version: "4.3.1", Source: SourceRegistry,
				Artifacts: []Artifact{
					{Platform: "linux", Arch: "amd64", URL: "https://example.com/cmake-linux.tar.gz", Checksum: "sha256:c1", Size: 7, Strip: 1},
					{Platform: "darwin", Arch: "arm64", URL: "https://example.com/cmake-mac.tar.gz", Checksum: "sha256:c2", Size: 8, Strip: 1, LinkBinFrom: "CMake.app/Contents/bin"},
				},
			},
		},
		Packages: PackagesSection{
			Lockfiles: []EcoLockRef{
				{Ecosystem: "vcpkg", Path: "vcpkg.json", Digest: "sha256:v", Baseline: "abc123"},
				{Ecosystem: "pip", Path: "requirements.txt", Digest: "sha256:r"},
			},
		},
	}
}

// TestWriteRoundTrip is the core correctness check: anything Write puts on disk
// must load back into an equivalent Lockfile. Write sorts lf in place, so after
// the call lf holds the same canonical ordering that was serialized — meaning a
// reload should be deeply equal to it.
func TestWriteRoundTrip(t *testing.T) {
	lf, err := Load(filepath.Join("testdata", "golden.toml"))
	if err != nil {
		t.Fatalf("Load(golden): %v", err)
	}

	path := filepath.Join(t.TempDir(), "compendium.lock")
	if err = Write(path, lf); err != nil { // Write sorts *lf in place
		t.Fatalf("Write: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load(written): %v", err)
	}

	if !reflect.DeepEqual(lf, got) {
		t.Errorf("round-trip mismatch:\n in: %+v\nout: %+v", lf, got)
	}
}

// TestWriteDeterministic proves the sort makes output order-independent:
// identical data presented in two different slice orders must serialize to
// byte-identical files. This is ION-10's central promise.
func TestWriteDeterministic(t *testing.T) {
	dir := t.TempDir()

	lf := unsortedLockfile()
	first := filepath.Join(dir, "first.lock")
	if err := Write(first, &lf); err != nil {
		t.Fatalf("Write(first): %v", err)
	}

	// Reverse every slice. The data is unchanged, only the order differs, so
	// Write's sort must yield the exact same bytes.
	slices.Reverse(lf.Languages)
	slices.Reverse(lf.Tools)
	for i := range lf.Languages {
		slices.Reverse(lf.Languages[i].Artifacts)
	}
	for i := range lf.Tools {
		slices.Reverse(lf.Tools[i].Artifacts)
	}
	slices.Reverse(lf.Packages.Lockfiles)

	second := filepath.Join(dir, "second.lock")
	if err := Write(second, &lf); err != nil {
		t.Fatalf("Write(second): %v", err)
	}

	a, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Errorf("non-deterministic output:\n--- first ---\n%s\n--- second ---\n%s", a, b)
	}
}

// TestWriteHeader checks the AUTO-GENERATED banner is the first thing in the file.
func TestWriteHeader(t *testing.T) {
	lf := unsortedLockfile()
	path := filepath.Join(t.TempDir(), "compendium.lock")
	if err := Write(path, &lf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte(writeHeader)) {
		t.Errorf("missing header banner; file starts with:\n%s", firstLines(data, 2))
	}
}

// TestWriteAtomic checks the temp-file + rename dance: a pre-existing file is
// cleanly replaced and no temp file is left behind on success.
func TestWriteAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compendium.lock")

	// A stale file at the destination must be fully replaced, not appended to.
	if err := os.WriteFile(path, []byte("stale = contents\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	lf := unsortedLockfile()
	if err := Write(path, &lf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Stale content is gone: the file reloads and validates cleanly.
	if _, err := Load(path); err != nil {
		t.Fatalf("written file does not reload: %v", err)
	}

	// The temp file (.compendium.lock-*) must not survive a successful write.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".compendium.lock-") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("dir has %d entries, want 1 (just the lockfile)", len(entries))
	}
}

// firstLines returns the first n lines of b, for readable failure messages.
func firstLines(b []byte, n int) string {
	lines := strings.SplitN(string(b), "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
