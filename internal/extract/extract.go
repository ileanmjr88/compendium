package extract

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

func Extract(filePath, destDir string, strip int) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	var reader io.Reader
	switch {
	case strings.HasSuffix(filePath, ".tar.gz") || strings.HasSuffix(filePath, ".tgz"):
		gz, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("extracting file: %w", err)
		}
		defer func() { _ = gz.Close() }()
		reader = gz
	case strings.HasSuffix(filePath, ".tar.xz"):
		xzReader, err := xz.NewReader(f)
		if err != nil {
			return fmt.Errorf("extracting xz file: %w", err)
		}
		reader = xzReader
	case strings.HasSuffix(filePath, ".tar.zst"):
		zstReader, err := zstd.NewReader(f)
		if err != nil {
			return fmt.Errorf("extracting zstd file: %w", err)
		}
		defer zstReader.Close()
		reader = zstReader
	case strings.HasSuffix(filePath, ".tar.bz2"):
		bz := bzip2.NewReader(f)
		reader = bz
	case strings.HasSuffix(filePath, ".zip"):
		_ = f.Close()
		return unzip(filePath, destDir, strip)
	default:
		return fmt.Errorf("unsupported format: %s", filePath)
	}
	return untar(reader, destDir, strip)
}

func untar(r io.Reader, destDir string, strip int) error {
	t := tar.NewReader(r)
	for {
		header, err := t.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("untar: %w", err)
		}

		// Strip leading path components
		name := header.Name
		if strip > 0 {
			for i := 0; i < strip; i++ {
				slash := strings.Index(name, "/")
				if slash == -1 {
					name = ""
					break
				}
				name = name[slash+1:]
			}
		}
		if name == "" {
			continue
		}

		target := filepath.Join(destDir, name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("creating directory: %w", err)
			}
		case tar.TypeReg:
			// ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("creating parent directory: %w", err)
			}
			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("creating file: %w", err)
			}
			if _, err := io.Copy(outFile, t); err != nil {
				_ = outFile.Close()
				return fmt.Errorf("writing file: %w", err)
			}
			_ = outFile.Chmod(header.FileInfo().Mode())
			_ = outFile.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("creating symlink parent directory: %w", err)
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("creating symlink: %w", err)
			}
		}
	}
	return nil
}

func unzip(filePath, destDir string, strip int) error {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return fmt.Errorf("opening zip: %w", err)
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		name := f.Name
		if strip > 0 {
			for i := 0; i < strip; i++ {
				slash := strings.Index(name, "/")
				if slash == -1 {
					name = ""
					break
				}
				name = name[slash+1:]
			}
		}
		if name == "" {
			continue
		}

		target := filepath.Join(destDir, name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("creating directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("creating parent directory: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("opening zip entry: %w", err)
		}

		outFile, err := os.Create(target)
		if err != nil {
			_ = rc.Close()
			return fmt.Errorf("creating file: %w", err)
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			_ = outFile.Close()
			_ = rc.Close()
			return fmt.Errorf("writing file: %w", err)
		}

		_ = outFile.Chmod(f.Mode())
		_ = outFile.Close()
		_ = rc.Close()
	}
	return nil
}
