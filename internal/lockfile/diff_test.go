package lockfile

import (
	"slices"
	"testing"

	"github.com/ileanmjr88/compendium/internal/config"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		name           string
		intent         config.Config
		resolved       *Lockfile
		platform, arch string
		want           []Change
	}{
		{
			// Specs match AND the current platform is already covered → no change.
			name:     "no changes",
			platform: "linux",
			arch:     "amd64",
			intent: config.Config{
				Languages: config.Languages{"go": "1.22"},
				Tools:     config.Tools{"ripgrep": "14"},
				Packages:  config.Packages{"pip": "requirements.txt"},
			},
			resolved: &Lockfile{
				Languages: []Entry{{Name: "go", Spec: "1.22", Artifacts: []Artifact{{Platform: "linux", Arch: "amd64"}}}},
				Tools:     []Entry{{Name: "ripgrep", Spec: "14", Artifacts: []Artifact{{Platform: "linux", Arch: "amd64"}}}},
				Packages:  PackagesSection{Lockfiles: []EcoLockRef{{Ecosystem: "pip", Path: "requirements.txt"}}},
			},
			want: nil,
		},
		{
			// Spec matches but the lock has no artifact for this platform → union it in.
			name:     "language platform missing",
			platform: "linux",
			arch:     "amd64",
			intent:   config.Config{Languages: config.Languages{"go": "1.22"}},
			resolved: &Lockfile{Languages: []Entry{{Name: "go", Spec: "1.22", Artifacts: []Artifact{{Platform: "darwin", Arch: "arm64"}}}}},
			want:     []Change{{Name: "go", Section: SectionLanguage, Kind: PlatformMissing, NewSpec: "1.22"}},
		},
		{
			name:     "language added",
			intent:   config.Config{Languages: config.Languages{"go": "1.22"}},
			resolved: &Lockfile{},
			want:     []Change{{Name: "go", Section: SectionLanguage, Kind: Added, NewSpec: "1.22"}},
		},
		{
			name:     "language removed",
			intent:   config.Config{},
			resolved: &Lockfile{Languages: []Entry{{Name: "go", Spec: "1.22"}}},
			want:     []Change{{Name: "go", Section: SectionLanguage, Kind: Removed, OldSpec: "1.22"}},
		},
		{
			name:     "language spec changed",
			intent:   config.Config{Languages: config.Languages{"go": "1.23"}},
			resolved: &Lockfile{Languages: []Entry{{Name: "go", Spec: "1.22"}}},
			want:     []Change{{Name: "go", Section: SectionLanguage, Kind: SpecChanged, OldSpec: "1.22", NewSpec: "1.23"}},
		},
		{
			name:     "tool added",
			intent:   config.Config{Tools: config.Tools{"ripgrep": "14"}},
			resolved: &Lockfile{},
			want:     []Change{{Name: "ripgrep", Section: SectionTool, Kind: Added, NewSpec: "14"}},
		},
		{
			name:     "package added",
			intent:   config.Config{Packages: config.Packages{"npm": "package.json"}},
			resolved: &Lockfile{},
			want:     []Change{{Name: "npm", Section: SectionPackage, Kind: Added, NewSpec: "package.json"}},
		},
		{
			name:     "package path changed",
			intent:   config.Config{Packages: config.Packages{"pip": "requirements-dev.txt"}},
			resolved: &Lockfile{Packages: PackagesSection{Lockfiles: []EcoLockRef{{Ecosystem: "pip", Path: "requirements.txt"}}}},
			want:     []Change{{Name: "pip", Section: SectionPackage, Kind: SpecChanged, OldSpec: "requirements.txt", NewSpec: "requirements-dev.txt"}},
		},
		{
			// Digest drift (same path, changed Digest) must NOT surface — Diff is pure.
			name:     "package digest drift ignored",
			intent:   config.Config{Packages: config.Packages{"pip": "requirements.txt"}},
			resolved: &Lockfile{Packages: PackagesSection{Lockfiles: []EcoLockRef{{Ecosystem: "pip", Path: "requirements.txt", Digest: "sha256:abc"}}}},
			want:     nil,
		},
		{
			// Output must be sorted by (Section, Name): languages, then tools, then
			// packages; alphabetical within each section.
			name: "sorted across sections",
			intent: config.Config{
				Languages: config.Languages{"rust": "1", "go": "1"},
				Tools:     config.Tools{"ripgrep": "14"},
				Packages:  config.Packages{"pip": "r.txt"},
			},
			resolved: &Lockfile{},
			want: []Change{
				{Name: "go", Section: SectionLanguage, Kind: Added, NewSpec: "1"},
				{Name: "rust", Section: SectionLanguage, Kind: Added, NewSpec: "1"},
				{Name: "ripgrep", Section: SectionTool, Kind: Added, NewSpec: "14"},
				{Name: "pip", Section: SectionPackage, Kind: Added, NewSpec: "r.txt"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Diff(tt.intent, tt.resolved, tt.platform, tt.arch)
			if !slices.Equal(got, tt.want) {
				t.Errorf("Diff() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
