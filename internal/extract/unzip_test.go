package extract

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestUnzip_DirectoryEntries(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.zip")

	zf, _ := os.Create(archivePath)
	zw := zip.NewWriter(zf)
	// Explicit directory entry (trailing slash signals directory)
	zw.Create("bin/")
	w, _ := zw.Create("bin/tool")
	w.Write([]byte("tool content"))
	zw.Close()
	zf.Close()

	destDir := filepath.Join(tmpDir, "output")
	if err := Extract(archivePath, destDir, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(filepath.Join(destDir, "bin"))
	if err != nil {
		t.Fatalf("bin directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected bin to be a directory")
	}

	if _, err := os.Stat(filepath.Join(destDir, "bin", "tool")); err != nil {
		t.Errorf("bin/tool not extracted: %v", err)
	}
}

func TestUnzip_NestedFileCreatesParents(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.zip")

	// File in nested path with no explicit directory entries.
	zf, _ := os.Create(archivePath)
	zw := zip.NewWriter(zf)
	w, _ := zw.Create("a/b/c/deep.txt")
	w.Write([]byte("deep"))
	zw.Close()
	zf.Close()

	destDir := filepath.Join(tmpDir, "output")
	if err := Extract(archivePath, destDir, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(destDir, "a", "b", "c", "deep.txt"))
	if err != nil {
		t.Fatalf("nested file not extracted: %v", err)
	}
	if string(got) != "deep" {
		t.Errorf("content mismatch: got %q", string(got))
	}
}

func TestUnzip_StripSkipsShallowEntries(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.zip")

	zf, _ := os.Create(archivePath)
	zw := zip.NewWriter(zf)
	// "toplevel" has no slash — with strip=1 it should be skipped (name="")
	w1, _ := zw.Create("toplevel")
	w1.Write([]byte("skip me"))
	// "root/keep.txt" with strip=1 becomes "keep.txt"
	w2, _ := zw.Create("root/keep.txt")
	w2.Write([]byte("keep"))
	zw.Close()
	zf.Close()

	destDir := filepath.Join(tmpDir, "output")
	if err := Extract(archivePath, destDir, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(destDir, "toplevel")); !os.IsNotExist(err) {
		t.Error("toplevel should have been stripped away")
	}
	if _, err := os.Stat(filepath.Join(destDir, "keep.txt")); err != nil {
		t.Errorf("keep.txt not extracted: %v", err)
	}
}

func TestUnzip_CorruptArchiveReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "bad.zip")
	os.WriteFile(archivePath, []byte("not a real zip"), 0644)

	err := Extract(archivePath, filepath.Join(tmpDir, "output"), 0)
	if err == nil {
		t.Error("expected error for corrupt zip, got nil")
	}
}

func TestUnzip_PreservesExecutableMode(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.zip")

	zf, _ := os.Create(archivePath)
	zw := zip.NewWriter(zf)
	header := &zip.FileHeader{Name: "run.sh", Method: zip.Deflate}
	header.SetMode(0755)
	w, _ := zw.CreateHeader(header)
	w.Write([]byte("#!/bin/sh"))
	zw.Close()
	zf.Close()

	destDir := filepath.Join(tmpDir, "output")
	if err := Extract(archivePath, destDir, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(filepath.Join(destDir, "run.sh"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Errorf("expected executable bit preserved, got %v", info.Mode())
	}
}
