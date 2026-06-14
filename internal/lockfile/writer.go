package lockfile

import (
	"bytes"
	"cmp"
	"os"
	"path/filepath"
	"slices"

	"github.com/pelletier/go-toml/v2"
)

func Write(path string, lf *Lockfile) error {
	slices.SortFunc(lf.Languages, func(a, b Entry) int {
		return cmp.Compare(a.Name, b.Name)
	})

	slices.SortFunc(lf.Tools, func(a, b Entry) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for i := range lf.Languages {
		slices.SortFunc(lf.Languages[i].Artifacts, func(a, b Artifact) int {
			return cmp.Or(cmp.Compare(a.Platform, b.Platform), cmp.Compare(a.Arch, b.Arch))
		})
	}
	for i := range lf.Tools {
		slices.SortFunc(lf.Tools[i].Artifacts, func(a, b Artifact) int {
			return cmp.Or(cmp.Compare(a.Platform, b.Platform), cmp.Compare(a.Arch, b.Arch))
		})
	}

	slices.SortFunc(lf.Packages.Lockfiles, func(a, b EcoLockRef) int {
		return cmp.Compare(a.Ecosystem, b.Ecosystem)
	})

	var buf bytes.Buffer
	buf.WriteString("# compendium.lock\n# AUTO-GENERATED. Do not edit by hand. Commit to version control.\n")
	enc := toml.NewEncoder(&buf)
	enc.SetIndentTables(true)
	if err := enc.Encode(lf); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".compendium.lock-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), path)
}
