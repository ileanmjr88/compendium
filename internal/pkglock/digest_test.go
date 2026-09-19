package pkglock

import (
	"strings"
	"testing"
)

// The two pinned values below are reproducible without running this code:
//
//	shasum -a 256 testdata/vcpkg-alone/vcpkg.json
//	  -> 87eae5c0...c1a1f
//
//	d1=$(shasum -a 256 vcpkg.json | cut -d' ' -f1)
//	d2=$(shasum -a 256 vcpkg-configuration.json | cut -d' ' -f1)
//	printf "%s%s" "$d1" "$d2" | xxd -r -p | shasum -a 256
//	  -> b6e46ef9...61e7
//
// That external reproducibility is the point: ION-20 §2 requires a third
// party to be able to recompute a digest independently, so these assert
// agreement with a standard, not with ourselves. They also have to match the
// values recorded in internal/lockfile/testdata/golden.toml.
//
// Both fixtures are LF-only with no trailing whitespace and a single final
// newline, so normalize is a no-op on them — which is why plain shasum on the
// raw bytes agrees with sha256(normalize(bytes)).
const (
	digestVcpkgAlone   = "sha256:87eae5c0f15eb4e05a47689e64da9396257cc51709e1521ff2217280d95c1a1f"
	digestVcpkgSibling = "sha256:b6e46ef9e1696a44b8ec3073cd56221e9fcbfa6ca2820eeaefe0f0dd534c61e7"
)

func TestDigest(t *testing.T) {
	tests := []struct {
		name  string
		root  string
		paths []string
		want  string
	}{
		// One covered file: the digest IS that file's hash. Running the
		// multi-file composition over a single input would yield
		// sha256(d1) = e3d634d6..., which matches neither golden.toml nor
		// anything a user could reproduce with shasum.
		{
			"single file is not a hash-of-hash",
			"testdata/vcpkg-alone",
			[]string{"vcpkg.json"},
			digestVcpkgAlone,
		},
		// Two covered files: sha256 over the concatenated raw 32-byte
		// digests, in cover's order.
		{
			"two files compose",
			"testdata/vcpkg-sibling",
			[]string{"vcpkg.json", "vcpkg-configuration.json"},
			digestVcpkgSibling,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := digest(tt.root, tt.paths)
			if err != nil {
				t.Fatalf("digest(%q, %v): unexpected error: %v", tt.root, tt.paths, err)
			}
			if got != tt.want {
				t.Errorf("digest(%q, %v)\n got %s\nwant %s", tt.root, tt.paths, got, tt.want)
			}
		})
	}
}

// TestDigestThroughCover proves the two halves agree: whatever cover decides
// to span, digest hashes. The nested fixture is a copy of vcpkg-sibling, so
// the composed value must be identical despite the different declared path.
func TestDigestThroughCover(t *testing.T) {
	const root = "testdata/vcpkg-nested"

	paths, err := cover(root, "deps/vcpkg.json")
	if err != nil {
		t.Fatalf("cover: %v", err)
	}

	got, err := digest(root, paths)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	if got != digestVcpkgSibling {
		t.Errorf("covers=%v\n got %s\nwant %s", paths, got, digestVcpkgSibling)
	}
}

// TestDigestErrors covers the paths that must fail loudly. A digest computed
// over fewer files than cover reported would look valid while silently
// spanning a subset — the exact false negative the sibling rule exists to
// prevent — so neither case may return a digest.
func TestDigestErrors(t *testing.T) {
	tests := []struct {
		name  string
		root  string
		paths []string
	}{
		{"no paths", "testdata", nil},
		{"missing file", "testdata/vcpkg-alone", []string{"nope.json"}},
		{"one of two missing", "testdata/vcpkg-alone", []string{"vcpkg.json", "nope.json"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := digest(tt.root, tt.paths)
			if err == nil {
				t.Fatalf("digest(%q, %v) = %q, want an error", tt.root, tt.paths, got)
			}
			if got != "" {
				t.Errorf("digest returned %q alongside an error, want empty", got)
			}
		})
	}
}

// TestDigestFormat guards the wire format. The lockfile validator rejects a
// checksum without the sha256: prefix, and the value is compared as a string
// against what is already stored in compendium.lock.
func TestDigestFormat(t *testing.T) {
	got, err := digest("testdata/vcpkg-alone", []string{"vcpkg.json"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "sha256:") {
		t.Errorf("digest = %q, want a sha256: prefix", got)
	}
	if hex := strings.TrimPrefix(got, "sha256:"); len(hex) != 64 {
		t.Errorf("hex portion is %d chars, want 64", len(hex))
	}
	if got != strings.ToLower(got) {
		t.Errorf("digest = %q, want lowercase hex", got)
	}
}
