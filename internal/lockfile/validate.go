package lockfile

import (
	"fmt"
	"strconv"
	"strings"
)

var knownLockPlatforms = map[string]bool{
	"linux-amd64": true, "linux-arm64": true,
	"darwin-amd64": true, "darwin-arm64": true,
}

func (l *Lockfile) Validate() error {
	if l.Meta.Schema != CurrentSchema {
		return &ValidationError{
			Field:   "meta.schema",
			Value:   strconv.Itoa(l.Meta.Schema),
			Message: fmt.Sprintf("schema must equal %d", CurrentSchema),
		}
	}

	if len(l.Meta.Platforms) == 0 {
		return &ValidationError{
			Field:   "meta.platforms",
			Message: "must list at least one platform",
		}
	}
	for _, p := range l.Meta.Platforms {
		if !knownLockPlatforms[p] {
			return &ValidationError{
				Field:   "meta.platforms",
				Value:   p,
				Message: "unknown platform",
			}
		}
	}

	if err := validateEntries("languages", l.Languages); err != nil {
		return err
	}
	if err := validateEntries("tool", l.Tools); err != nil {
		return err
	}

	seenEco := map[string]bool{}
	for _, ref := range l.Packages.Lockfiles {
		if seenEco[ref.Ecosystem] {
			return &ValidationError{
				Field:   "packages.lockfile.ecosystem",
				Value:   ref.Ecosystem,
				Message: "duplicate ecosystem",
			}
		}
		seenEco[ref.Ecosystem] = true
	}
	return nil
}

func validateEntries(kind string, entries []Entry) error {
	names := map[string]bool{}
	for _, e := range entries {
		if names[e.Name] {
			return &ValidationError{
				Field:   kind + ".name",
				Value:   e.Name,
				Message: "duplicate entry name",
			}
		}
		names[e.Name] = true

		if e.Source != SourceRegistry {
			return &ValidationError{
				Field:   fmt.Sprintf("%s[%s.source", kind, e.Name),
				Value:   e.Source,
				Message: fmt.Sprintf("source must be %q", SourceRegistry),
			}
		}

		seenPA := map[string]bool{}
		for _, a := range e.Artifacts {
			field := fmt.Sprintf("%s[%s].artifacts[%s-%s]", kind, e.Name, a.Platform, a.Arch)

			if a.URL == "" {
				return &ValidationError{
					Field:   field + "url",
					Message: "artifact url is empty",
				}
			}
			if !strings.HasPrefix(a.Checksum, "sha256:") {
				return &ValidationError{
					Field:   field + ".checksum",
					Value:   a.Checksum,
					Message: "checksum must have sha256: prefix",
				}
			}
			if a.Size == 0 {
				return &ValidationError{
					Field:   field + ".size",
					Message: "artifact size must be non-zero",
				}
			}

			pa := a.Platform + "-" + a.Arch
			if seenPA[pa] {
				return &ValidationError{
					Field:   field,
					Value:   pa,
					Message: "duplicate (platform, arch) in artifacts",
				}
			}
			seenPA[pa] = true
		}
	}
	return nil
}
