package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallBinary(t *testing.T) {
	t.Run("copies file and marks executable", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "src-bin")
		if err := os.WriteFile(src, []byte("fake binary"), 0644); err != nil {
			t.Fatalf("write src: %v", err)
		}

		destDir := filepath.Join(tmp, "dest")
		if err := installBinary(src, destDir, "mytool"); err != nil {
			t.Fatalf("installBinary: %v", err)
		}

		dest := filepath.Join(destDir, "bin", "mytool")
		info, err := os.Stat(dest)
		if err != nil {
			t.Fatalf("stat dest: %v", err)
		}
		if info.Mode()&0111 == 0 {
			t.Errorf("expected executable bit set, got mode %v", info.Mode())
		}

		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("read dest: %v", err)
		}
		if string(got) != "fake binary" {
			t.Errorf("content mismatch: got %q", string(got))
		}
	})

	t.Run("missing source returns error", func(t *testing.T) {
		tmp := t.TempDir()
		err := installBinary(filepath.Join(tmp, "nope"), filepath.Join(tmp, "dest"), "mytool")
		if err == nil {
			t.Error("expected error for missing source, got nil")
		}
	})

	t.Run("unwritable dest returns error", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "src-bin")
		os.WriteFile(src, []byte("x"), 0644)

		// Point destDir at a path where MkdirAll will fail: inside a regular file.
		blocker := filepath.Join(tmp, "blocker")
		os.WriteFile(blocker, []byte("x"), 0644)

		err := installBinary(src, filepath.Join(blocker, "sub"), "mytool")
		if err == nil {
			t.Error("expected error when destDir is under a file, got nil")
		}
	})
}
