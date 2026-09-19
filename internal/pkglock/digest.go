package pkglock

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

func fileDigest(path string) ([32]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return [32]byte{}, fmt.Errorf("reading %s: %w", path, err)
	}

	return sha256.Sum256(normalize(raw)), nil
}

func digest(root string, paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("no paths given")
	}

	var digests [][32]byte
	for _, p := range paths {
		d, err := fileDigest(filepath.Join(root, p))
		if err != nil {
			return "", err
		}
		digests = append(digests, d)
	}

	if len(digests) == 1 {
		return fmt.Sprintf("sha256:%x", digests[0]), nil
	}

	h := sha256.New()
	for _, d := range digests {
		h.Write(d[:])
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
