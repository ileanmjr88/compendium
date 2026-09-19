package pkglock

import (
	"fmt"
)

type Result struct {
	Digest string   // "sha256:<hex>"
	Covers []string // project-root-relative, hash-composition order
}

func Resolve(root, ecosystem, declared string) (Result, error) {
	paths, err := cover(root, declared)
	if err != nil {
		return Result{}, fmt.Errorf("package %q: covering %q: %w", ecosystem, declared, err)
	}

	if err := preflight(root, ecosystem, paths); err != nil {
		return Result{}, err
	}

	digest, err := digest(root, paths)
	if err != nil {
		return Result{}, fmt.Errorf("package %q: %w", ecosystem, err)
	}

	return Result{Digest: digest, Covers: paths}, nil
}
