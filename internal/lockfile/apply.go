package lockfile

import (
	"slices"
)

type Resolved struct {
	Languages map[string]Entry
	Tools     map[string]Entry
	Packages  map[string]EcoLockRef
}

func (l *Lockfile) Apply(changes []Change, resolved Resolved) {
	for _, c := range changes {
		switch c.Section {
		case SectionLanguage:
			l.Languages = applyEntry(l.Languages, c, resolved.Languages)
			l.Meta.Platforms = addPlatforms(l.Meta.Platforms, resolved.Languages[c.Name])
		case SectionTool:
			l.Tools = applyEntry(l.Tools, c, resolved.Tools)
			l.Meta.Platforms = addPlatforms(l.Meta.Platforms, resolved.Tools[c.Name])
		case SectionPackage:
			l.Packages.Lockfiles = applyPkg(l.Packages.Lockfiles, c, resolved.Packages)
		}
	}
}

func applyEntry(entries []Entry, c Change, resolved map[string]Entry) []Entry {
	// Remove -> drop the entry whose Name == c.Name
	if c.Kind == Removed {
		return slices.DeleteFunc(entries, func(e Entry) bool {
			return e.Name == c.Name
		})
	}

	next := resolved[c.Name]
	i := slices.IndexFunc(entries, func(e Entry) bool {
		return e.Name == c.Name
	})

	// PlatformMissing -> the entry already exists at this spec; just merge the
	// newly resolved platform's artifact(s) into it. Replacing the whole entry
	// (below) would drop the other platforms' artifacts we want to keep.
	if c.Kind == PlatformMissing && i >= 0 {
		entries[i].Artifacts = upsertArtifacts(entries[i].Artifacts, next.Artifacts)
		return entries
	}

	// Added or SpecChanged -> replace if present, else append.
	if i >= 0 {
		entries[i] = next
		return entries
	}
	return append(entries, next)
}

func applyPkg(refs []EcoLockRef, c Change, resolved map[string]EcoLockRef) []EcoLockRef {
	// Remove
	if c.Kind == Removed {
		return slices.DeleteFunc(refs, func(e EcoLockRef) bool {
			return e.Ecosystem == c.Name
		})
	}

	next := resolved[c.Name]
	if i := slices.IndexFunc(refs, func(e EcoLockRef) bool {
		return e.Ecosystem == c.Name
	}); i >= 0 {
		refs[i] = next
		return refs
	}
	return append(refs, next)
}

func upsertArtifacts(existing, incoming []Artifact) []Artifact {
	for _, a := range incoming {
		if j := slices.IndexFunc(existing, func(e Artifact) bool {
			return e.Platform == a.Platform && e.Arch == a.Arch
		}); j >= 0 {
			existing[j] = a
		} else {
			existing = append(existing, a)
		}
	}
	return existing
}

func addPlatforms(platforms []string, e Entry) []string {
	for _, a := range e.Artifacts {
		p := a.Platform + "-" + a.Arch
		if !slices.Contains(platforms, p) {
			platforms = append(platforms, p)
		}
	}
	return platforms
}
