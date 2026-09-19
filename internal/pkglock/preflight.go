package pkglock

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// preflight verifies every file the digest for one [packages] entry will
// cover: it exists, it is a regular file, and it can actually be opened.
//
// Every failure is collected rather than returned on sight, so one run
// reports all the problems across all covered files instead of making the
// user fix them one at a time. errors.Join returns nil for an empty slice,
// so the happy path needs no special case — and because Join also skips nil
// arguments, callers can accumulate across [packages] entries the same way.
//
// Errors are returned, never printed: rendering belongs to the CLI layer,
// which owns the ui package.
func preflight(root, ecosystem string, paths []string) error {
	var errs []error

	for _, p := range paths {
		abs := filepath.Join(root, p)

		info, err := os.Stat(abs)
		if errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, fmt.Errorf(
				"package %q references lockfile %q but it was not found\n"+
					"  expected at: %s\n"+
					"  hint: generate it, or remove the entry from [packages]",
				ecosystem, p, abs))
			continue
		} else if err != nil {
			errs = append(errs, fmt.Errorf("package %q: checking %s: %w", ecosystem, p, err))
			continue
		} else if !info.Mode().IsRegular() {
			errs = append(errs, fmt.Errorf(
				"package %q references lockfile %q but it is not a regular file\n"+
					"  found at: %s",
				ecosystem, p, abs))
			// Without this, the Open below would succeed on a directory and
			// report a second, more confusing error for the same path.
			continue
		}

		// Stat cannot answer "is this readable" — permission bits lie once
		// root, ACLs or mount options are involved. Opening it is the only
		// honest check.
		f, err := os.Open(abs)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"package %q references lockfile %q but it is not readable\n"+
					"  found at: %s\n"+
					"  hint: %v",
				ecosystem, p, abs, err))
			continue
		}
		_ = f.Close()
	}

	return errors.Join(errs...)
}
