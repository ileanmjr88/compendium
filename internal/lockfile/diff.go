package lockfile

import (
	"cmp"
	"slices"

	"github.com/ileanmjr88/compendium/internal/config"
)

type Section int

const (
	SectionLanguage Section = iota // 0
	SectionTool                    // 1
	SectionPackage                 // 2
)

type ChangeKind int

const (
	Added           ChangeKind = iota // 0
	Removed                           // 1
	SpecChanged                       // 2
	PlatformMissing                   // 3
)

type Change struct {
	Name    string
	Section Section
	Kind    ChangeKind
	OldSpec string
	NewSpec string
}

func Diff(intent config.Config, resolved *Lockfile, platform string, arch string) []Change {
	var changes []Change
	changes = append(changes, diffSection(SectionLanguage, intent.Languages, resolved.Languages, platform, arch)...)
	changes = append(changes, diffSection(SectionTool, intent.Tools, resolved.Tools, platform, arch)...)
	changes = append(changes, diffPackages(intent.Packages, resolved.Packages.Lockfiles)...)

	// Deterministic order: sort by (Section, Name). Section is an int, so the
	// sections fall out as language < tool < package.
	slices.SortFunc(changes, func(a, b Change) int {
		return cmp.Or(cmp.Compare(a.Section, b.Section), cmp.Compare(a.Name, b.Name))
	})

	return changes
}

func diffSection(selection Section, intent map[string]string, locked []Entry, platform, arch string) []Change {
	lockByName := make(map[string]Entry, len(locked))
	for _, e := range locked {
		lockByName[e.Name] = e
	}

	var changes []Change

	// Pass 1 — walk the toml (intent) map.
	// not in lock  -> Added (only NewSpec)
	// in lock, spec differs -> SpecChanged (both specs)
	// in lock, spec equal -> no change
	for name, tomlSpec := range intent {
		e, ok := lockByName[name]
		switch {
		case !ok:
			changes = append(changes, Change{Name: name, Section: selection, Kind: Added, NewSpec: tomlSpec})
		case tomlSpec != e.Spec:
			changes = append(changes, Change{Name: name, Section: selection, Kind: SpecChanged, OldSpec: e.Spec, NewSpec: tomlSpec})
		case !hasArtifact(e, platform, arch):
			changes = append(changes, Change{Name: name, Section: selection, Kind: PlatformMissing, NewSpec: e.Spec})
		}
	}

	// Pass 2 — walk `locked`: name not in `intent` -> Removed (OldSpec only).
	for name, e := range lockByName {
		if _, ok := intent[name]; !ok {
			changes = append(changes, Change{Name: name, Section: selection, Kind: Removed, OldSpec: e.Spec})
		}
	}

	return changes
}

// diffPackages compares the toml [packages] map (ecosystem -> path) against the
// lock's resolved ecosystem lockfile refs. It is keyed by ecosystem, and the
// "spec" being compared is the manifest path.
//
// It deliberately does NOT detect digest drift (the lock's content hash): that
// would require reading and hashing files on disk, and Diff must stay pure. A
// changed lockfile with an unchanged path produces no Change here — catching
// that drift is the installer's job, not Diff's.
func diffPackages(intent config.Packages, locked []EcoLockRef) []Change {
	lockByEco := make(map[string]string, len(locked))
	for _, l := range locked {
		lockByEco[l.Ecosystem] = l.Path
	}

	var changes []Change

	// Pass 1 — walk the toml map (ecosystem -> path).
	for eco, path := range intent {
		lockPath, ok := lockByEco[eco]
		if !ok {
			changes = append(changes, Change{Name: eco, Section: SectionPackage, Kind: Added, NewSpec: path})
		} else if path != lockPath {
			changes = append(changes, Change{Name: eco, Section: SectionPackage, Kind: SpecChanged, OldSpec: lockPath, NewSpec: path})
		}
	}

	// Pass 2 — walk the lock: ecosystem not in the toml -> Removed.
	for eco, lockPath := range lockByEco {
		if _, ok := intent[eco]; !ok {
			changes = append(changes, Change{Name: eco, Section: SectionPackage, Kind: Removed, OldSpec: lockPath})
		}
	}

	return changes
}

func hasArtifact(e Entry, platform, arch string) bool {
	return slices.ContainsFunc(e.Artifacts, func(a Artifact) bool {
		return a.Platform == platform && a.Arch == arch
	})
}
