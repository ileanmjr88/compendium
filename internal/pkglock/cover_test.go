package pkglock

import (
	"slices"
	"testing"
)

// TestCover pins which files a [packages] entry's digest spans.
//
// The returned order is part of the frozen schema-v1 contract — it feeds the
// digest composition — so these cases assert the exact slice, never a set.
func TestCover(t *testing.T) {
	tests := []struct {
		name     string
		root     string
		declared string
		want     []string
	}{
		// The sibling rule: vcpkg-configuration.json can declare a
		// default-registry baseline that overrides builtin-baseline, so the
		// digest has to span both files when it is present.
		{
			"vcpkg with sibling",
			"testdata/vcpkg-sibling", "vcpkg.json",
			[]string{"vcpkg.json", "vcpkg-configuration.json"},
		},
		// Absent sibling is the normal single-file case, not an error.
		{
			"vcpkg without sibling",
			"testdata/vcpkg-alone", "vcpkg.json",
			[]string{"vcpkg.json"},
		},
		// The sibling is looked up next to the declared file, not at the
		// project root. At the top level both spellings coincide, so only a
		// nested path actually exercises this.
		{
			"vcpkg nested, sibling resolved alongside it",
			"testdata/vcpkg-nested", "deps/vcpkg.json",
			[]string{"deps/vcpkg.json", "deps/vcpkg-configuration.json"},
		},
		// Every other ecosystem is always exactly one file.
		{
			"non-vcpkg ecosystem",
			"testdata", "requirements.txt",
			[]string{"requirements.txt"},
		},
		// cover reports coverage, it does not validate. A missing declared
		// file must still come back so preflight can produce the real error;
		// dropping it here would silently digest nothing.
		{
			"missing declared file is still covered",
			"testdata/vcpkg-alone", "nope.json",
			[]string{"nope.json"},
		},
		{
			"missing root falls back to single file",
			"testdata/does-not-exist", "vcpkg.json",
			[]string{"vcpkg.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cover(tt.root, tt.declared)
			if err != nil {
				t.Fatalf("cover(%q, %q): unexpected error: %v", tt.root, tt.declared, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("cover(%q, %q) = %v, want %v", tt.root, tt.declared, got, tt.want)
			}
		})
	}
}
