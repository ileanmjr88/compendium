package extract

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

const testContent = "hello from compendium!"

func createTarContent(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	content := []byte(testContent)

	tw.WriteHeader(&tar.Header{
		Name: "hello.txt",
		Mode: 0644,
		Size: int64(len(content)),
	})

	tw.Write(content)
	tw.Close()
	return buf.Bytes()
}

func createTarContentWithSymlink(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	content := []byte(testContent)

	tw.WriteHeader(&tar.Header{
		Name:     "go/bin/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	})

	tw.WriteHeader(&tar.Header{
		Name: "go/bin/hello.txt",
		Mode: 0644,
		Size: int64(len(content)),
	})
	tw.Write(content)

	tw.WriteHeader(&tar.Header{
		Name:     "go/bin/link",
		Typeflag: tar.TypeSymlink,
		Linkname: "hello.txt",
	})

	tw.Close()
	return buf.Bytes()
}

func createTarContentNested(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	content := []byte(testContent)

	// Directory entry
	tw.WriteHeader(&tar.Header{
		Name:     "go/bin/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	})

	// file inside the directory
	tw.WriteHeader(&tar.Header{
		Name: "go/bin/hello.txt",
		Mode: 0644,
		Size: int64(len(content)),
	})

	tw.Write(content)
	tw.Close()
	return buf.Bytes()
}

func compressGzip(t *testing.T, tarData []byte, path string) {
	t.Helper()
	f, _ := os.Create(path)
	gz := gzip.NewWriter(f)
	gz.Write(tarData)
	gz.Close()
	f.Close()
}

func compressXz(t *testing.T, tarData []byte, path string) {
	t.Helper()
	f, _ := os.Create(path)
	xzWriter, err := xz.NewWriter(f)
	if err != nil {
		t.Fatalf("creating xz writer: %v", err)
	}
	xzWriter.Write(tarData)
	xzWriter.Close()
	f.Close()
}

func compressZstd(t *testing.T, tarData []byte, path string) {
	t.Helper()
	f, _ := os.Create(path)
	zstWriter, err := zstd.NewWriter(f)
	if err != nil {
		t.Fatalf("creating zstd writer: %v", err)
	}
	zstWriter.Write(tarData)
	zstWriter.Close()
	f.Close()
}

func compressBz2(t *testing.T, tarData []byte, path string) {
	t.Helper()
	tarPath := path[:len(path)-4] // strip ".bz2" to get ".tar"
	os.WriteFile(tarPath, tarData, 0644)

	cmd := exec.Command("bzip2", tarPath)
	if err := cmd.Run(); err != nil {
		t.Skip("bzip2 not available, skipping")
	}
}

func verifyExtractedFile(t *testing.T, destDir, filePath string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(destDir, filePath))
	if err != nil {
		t.Fatalf("expected file %s: %v", filePath, err)
	}
	if string(data) != testContent {
		t.Errorf("expected '%s', got '%s'", testContent, data)
	}
}

func TestExtract(t *testing.T) {
	formats := []struct {
		name     string
		filename string
		compress func(t *testing.T, tarData []byte, path string)
	}{
		{"tar.gz", "test.tar.gz", compressGzip},
		{"tar.xz", "test.tar.xz", compressXz},
		{"tar.zst", "test.tar.zst", compressZstd},
		{"tar.bz2", "test.tar.bz2", compressBz2},
	}

	for _, tc := range formats {
		t.Run(tc.name, func(t *testing.T) {
			tarData := createTarContent(t)
			tmpDir := t.TempDir()
			archivePath := filepath.Join(tmpDir, tc.filename)

			tc.compress(t, tarData, archivePath)

			destDir := filepath.Join(tmpDir, "output")
			err := Extract(archivePath, destDir, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			verifyExtractedFile(t, destDir, "hello.txt")
		})
	}

	t.Run("strip components", func(t *testing.T) {
		tarData := createTarContentNested(t)
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")

		compressGzip(t, tarData, archivePath)

		destDir := filepath.Join(tmpDir, "output")
		err := Extract(archivePath, destDir, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		verifyExtractedFile(t, destDir, filepath.Join("bin", "hello.txt"))
	})

	t.Run("file not found", func(t *testing.T) {
		destDir := t.TempDir()
		err := Extract("/nonexistent/file.tar.gz", destDir, 0)
		if err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.zip")
		os.WriteFile(archivePath, []byte("fake"), 0644)

		err := Extract(archivePath, tmpDir, 0)
		if err == nil {
			t.Error("expected error for unsupported format")
		}
	})

	t.Run("symlink extraction", func(t *testing.T) {
		tarData := createTarContentWithSymlink(t)
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")

		compressGzip(t, tarData, archivePath)

		destDir := filepath.Join(tmpDir, "output")
		err := Extract(archivePath, destDir, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the regular file exists
		verifyExtractedFile(t, destDir, filepath.Join("bin", "hello.txt"))

		// Verify the symlink exists and is actually a symlink
		linkPath := filepath.Join(destDir, "bin", "link")
		fi, err := os.Lstat(linkPath)
		if err != nil {
			t.Fatalf("expected symlink at %s: %v", linkPath, err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			t.Errorf("expected symlink, got %s", fi.Mode())
		}

		// Verify the symlink target
		target, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("reading symlink: %v", err)
		}
		if target != "hello.txt" {
			t.Errorf("expected symlink target 'hello.txt', got '%s'", target)
		}

		// Verify reading through the symlink returns the file content
		data, err := os.ReadFile(linkPath)
		if err != nil {
			t.Fatalf("reading through symlink: %v", err)
		}
		if string(data) != testContent {
			t.Errorf("expected '%s', got '%s'", testContent, string(data))
		}
	})

	t.Run("symlink error parent directory", func(t *testing.T) {
		// Create a tar with a symlink whose parent path collides with a regular file
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		// Write a regular file at "bin" so MkdirAll("bin/subdir") fails
		content := []byte("blocker")
		tw.WriteHeader(&tar.Header{
			Name: "bin",
			Mode: 0644,
			Size: int64(len(content)),
		})
		tw.Write(content)

		tw.WriteHeader(&tar.Header{
			Name:     "bin/subdir/link",
			Typeflag: tar.TypeSymlink,
			Linkname: "target",
		})
		tw.Close()

		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		compressGzip(t, buf.Bytes(), archivePath)

		destDir := filepath.Join(tmpDir, "output")
		err := Extract(archivePath, destDir, 0)
		if err == nil {
			t.Error("expected error creating symlink parent directory")
		}
	})

	t.Run("symlink error duplicate", func(t *testing.T) {
		// Create a tar with two identical symlinks so the second os.Symlink fails
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)

		tw.WriteHeader(&tar.Header{
			Name:     "link",
			Typeflag: tar.TypeSymlink,
			Linkname: "target",
		})
		tw.WriteHeader(&tar.Header{
			Name:     "link",
			Typeflag: tar.TypeSymlink,
			Linkname: "target",
		})
		tw.Close()

		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		compressGzip(t, buf.Bytes(), archivePath)

		destDir := filepath.Join(tmpDir, "output")
		err := Extract(archivePath, destDir, 0)
		if err == nil {
			t.Error("expected error creating duplicate symlink")
		}
	})

	t.Run("zip", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.zip")

		// Create a zip file with a single file
		zf, _ := os.Create(archivePath)
		zw := zip.NewWriter(zf)
		w, _ := zw.Create("hello.txt")
		w.Write([]byte(testContent))
		zw.Close()
		zf.Close()

		destDir := filepath.Join(tmpDir, "output")
		err := Extract(archivePath, destDir, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		verifyExtractedFile(t, destDir, "hello.txt")
	})

	t.Run("zip with strip", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.zip")

		zf, _ := os.Create(archivePath)
		zw := zip.NewWriter(zf)
		w, _ := zw.Create("ninja-linux/ninja")
		w.Write([]byte(testContent))
		zw.Close()
		zf.Close()

		destDir := filepath.Join(tmpDir, "output")
		err := Extract(archivePath, destDir, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		verifyExtractedFile(t, destDir, "ninja")
	})

	t.Run("strip removes all components", func(t *testing.T) {
		tarData := createTarContentNested(t)
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")

		compressGzip(t, tarData, archivePath)

		destDir := filepath.Join(tmpDir, "output")
		err := Extract(archivePath, destDir, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// destination should be empty — all paths were stripped away
		entries, _ := os.ReadDir(destDir)
		if len(entries) != 0 {
			t.Errorf("expected empty directory, got %d entries", len(entries))
		}
	})
}
