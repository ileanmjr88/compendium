package pkglock

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// vcpkg is the one ecosystem whose digest can span two files. vcpkg.json
// declares builtin-baseline, but a sibling vcpkg-configuration.json may
// declare a default-registry baseline that overrides it — so hashing only
// vcpkg.json would report "no change" while the resolved package versions
// moved. See ION-20 §3.
const (
	vcpkgManifest = "vcpkg.json"
	vcpkgConfig   = "vcpkg-configuration.json"
)

// cover reports every file the digest for one [packages] entry must span,
// project-root-relative, in hash-composition order.
//
// root is used only to reach the disk and never appears in the result:
// compendium.lock is committed and shared, so recorded paths have to be
// machine-independent.
//
// The declared file is always included, even when it is missing — preflight
// is what reports that, with an actionable error. Only the sibling's
// existence is checked here, because that is the one thing that changes the
// answer. An absent sibling is the normal single-file case.
//
// Order is [declared, sibling] and must never be sorted. It feeds the digest
// composition, which is frozen at lockfile schema v1.
func cover(root, declared string) ([]string, error) {
	paths := []string{declared}
	if filepath.Base(declared) != vcpkgManifest {
		return paths, nil
	}

	sibling := filepath.Join(filepath.Dir(declared), vcpkgConfig)

	if _, err := os.Stat(filepath.Join(root, sibling)); err == nil {
		paths = append(paths, sibling)
	} else if !errors.Is(err, fs.ErrNotExist) {
		// Not-exist is expected. Any other failure (permissions, I/O) would
		// silently yield a digest covering one file instead of two, which is
		// the false negative the sibling rule exists to prevent.
		return nil, err
	}

	return paths, nil
}
