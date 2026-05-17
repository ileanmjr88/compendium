package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
)

const testContentInstaller = "hello install cmd!"

func findItem(items []InstallItem, name string) *InstallItem {
	for _, item := range items {
		if item.Name == name {
			return &item
		}
	}
	return nil
}

func createTarContent(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	content := []byte(testContentInstaller)

	tw.WriteHeader(&tar.Header{
		Name: "hello.txt",
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

// registryFiles holds the JSON bodies served by startTestServer. Tests build
// these after the server starts so they can splice in the random server URL.
type registryFiles struct {
	index     string
	goFile    string
	cmakeFile string
}

func startTestServer(t *testing.T, files *registryFiles, tarballPath string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/index.json":
			w.Write([]byte(files.index))
		case "/languages/go.json":
			w.Write([]byte(files.goFile))
		case "/tools/cmake.json":
			w.Write([]byte(files.cmakeFile))
		case "/test.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			http.ServeFile(w, r, tarballPath)
		default:
			http.NotFound(w, r)
		}
	}))
}

func buildIndex() string {
	return `{
		"version": "2",
		"updated_at": "2026-03-28",
		"languages": {
			"go": {"latest":"1.22.0","versions":1,"file":"languages/go.json"}
		},
		"tools": {
			"cmake": {"latest":"3.28.1","versions":1,"file":"tools/cmake.json"}
		}
	}`
}

func buildGoFile(serverURL, checksum string, size int) string {
	return fmt.Sprintf(`{
		"1.22.0": {
			"%s": {
				"%s": {"url":"%s/test.tar.gz","checksum":"%s","size":%d,"strip":0}
			}
		}
	}`, runtime.GOOS, runtime.GOARCH, serverURL, checksum, size)
}

func buildCmakeFile(serverURL, checksum string, size int) string {
	return fmt.Sprintf(`{
		"3.28.1": {
			"%s": {
				"%s": {"url":"%s/test.tar.gz","checksum":"%s","size":%d,"strip":0}
			}
		}
	}`, runtime.GOOS, runtime.GOARCH, serverURL, checksum, size)
}

func buildCmakeFileWithLinkBin(serverURL, checksum string, size int, linkBinFrom string) string {
	return fmt.Sprintf(`{
		"3.28.1": {
			"%s": {
				"%s": {"url":"%s/test.tar.gz","checksum":"%s","size":%d,"strip":0,"link_bin_from":"%s"}
			}
		}
	}`, runtime.GOOS, runtime.GOARCH, serverURL, checksum, size, linkBinFrom)
}

type tarFile struct {
	name    string
	content []byte
	mode    int64
}

// createTarFromFiles produces an uncompressed tar archive containing the given
// regular files. Parent directories are written implicitly: untar's MkdirAll
// handles them at extraction time, matching how some real archives (CMake's
// .app bundle, etc.) lay out nested paths without explicit dir entries.
func createTarFromFiles(t *testing.T, files []tarFile) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, f := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name: f.name,
			Mode: f.mode,
			Size: int64(len(f.content)),
		}); err != nil {
			t.Fatalf("tar header %s: %v", f.name, err)
		}
		if _, err := tw.Write(f.content); err != nil {
			t.Fatalf("tar write %s: %v", f.name, err)
		}
	}
	tw.Close()
	return buf.Bytes()
}
