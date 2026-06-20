package lockfile

import (
	"slices"
	"testing"
)

func TestApplyAddsLanguage(t *testing.T) {
	lock := &Lockfile{}

	changes := []Change{
		{Name: "go", Section: SectionLanguage, Kind: Added, NewSpec: "1.22.2"},
	}
	resolved := Resolved{Languages: map[string]Entry{"go": {Name: "go", Version: "1.22.1", Spec: "1.22.2"}}}

	lock.Apply(changes, resolved)

	if len(lock.Languages) != 1 {
		t.Fatalf("want 1 language: %d", len(lock.Languages))
	}
	if got := lock.Languages[0]; got.Name != "go" || got.Spec != "1.22.2" {
		t.Errorf("got %+v", got)
	}
}

func TestApplyRemovesEntry(t *testing.T) {
	lock := &Lockfile{
		Languages: []Entry{{Name: "go", Spec: "1.22.2"}},
	}

	changes := []Change{
		{Name: "go", Section: SectionLanguage, Kind: Removed, OldSpec: "1.22.2"},
	}

	resolved := Resolved{}

	lock.Apply(changes, resolved)

	if len(lock.Languages) != 0 {
		t.Errorf("want 0 languages, got %d: %+v", len(lock.Languages), lock.Languages)
	}
}

func TestApplyReplaceChangedSpec(t *testing.T) {
	lock := &Lockfile{
		Languages: []Entry{{Name: "go", Spec: "1.22.1"}},
	}

	changed := []Change{
		{Name: "go", Section: SectionLanguage, Kind: SpecChanged, NewSpec: "1.24.2"},
	}

	resolved := Resolved{Languages: map[string]Entry{"go": {Name: "go", Version: "1.24.2", Spec: "1.24.2"}}}

	lock.Apply(changed, resolved)

	if len(lock.Languages) != 1 {
		t.Fatalf("want 1 language change: %d", len(lock.Languages))
	}

	if got := lock.Languages[0]; got.Name != "go" || got.Spec != "1.24.2" {
		t.Errorf("got %+v", got)
	}
}

func TestApplyLeavesUnchangedEntries(t *testing.T) {
	lock := &Lockfile{
		Languages: []Entry{{Name: "go", Spec: "1.22.1"}},
	}

	changes := []Change{}
	resolved := Resolved{}

	lock.Apply(changes, resolved)

	if len(lock.Languages) != 1 {
		t.Fatalf("want 1 langauge, got %d", len(lock.Languages))
	}

	if got := lock.Languages[0]; got.Name != "go" || got.Spec != "1.22.1" {
		t.Errorf("got %+v", got)
	}
}

func TestApplyUnionsPlatformMissing(t *testing.T) {
	lock := &Lockfile{
		Meta: Meta{Platforms: []string{"darwin-arm64"}},
		Languages: []Entry{{
			Name: "go", Spec: "1.26.1",
			Artifacts: []Artifact{{Platform: "darwin", Arch: "arm64"}},
		}},
	}

	changes := []Change{
		{Name: "go", Section: SectionLanguage, Kind: PlatformMissing, NewSpec: "1.26.1"},
	}

	resolved := Resolved{Languages: map[string]Entry{
		"go": {Name: "go", Spec: "1.26.1", Artifacts: []Artifact{{Platform: "linux", Arch: "amd64"}}},
	}}

	lock.Apply(changes, resolved)

	// The new platform's artifact is unioned in; the existing one is kept.
	got := lock.Languages[0]
	if len(got.Artifacts) != 2 {
		t.Fatalf("want 2 artifacts after union, got %d: %+v", len(got.Artifacts), got.Artifacts)
	}
	for _, want := range []Artifact{{Platform: "darwin", Arch: "arm64"}, {Platform: "linux", Arch: "amd64"}} {
		if !slices.ContainsFunc(got.Artifacts, func(a Artifact) bool {
			return a.Platform == want.Platform && a.Arch == want.Arch
		}) {
			t.Errorf("missing artifact %s-%s, got %+v", want.Platform, want.Arch, got.Artifacts)
		}
	}

	// meta.platforms grows to include the newly added platform (grow-only).
	if !slices.Contains(lock.Meta.Platforms, "linux-amd64") {
		t.Errorf("want meta.platforms to gain linux-amd64, got %v", lock.Meta.Platforms)
	}
	if !slices.Contains(lock.Meta.Platforms, "darwin-arm64") {
		t.Errorf("want meta.platforms to keep darwin-arm64, got %v", lock.Meta.Platforms)
	}
}
