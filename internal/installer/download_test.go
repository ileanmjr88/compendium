package installer

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ileanmjr88/compendium/internal/env"
	"github.com/ileanmjr88/compendium/internal/registry"
)

// newClientForTest constructs a *registry.Client pointed at the given test
// server. Test failures here are fatal — they indicate a broken fixture, not
// a bug in DownloadVerify.
func newClientForTest(t *testing.T, serverURL string) *registry.Client {
	t.Helper()
	c, err := registry.NewClient(serverURL + "/index.json")
	if err != nil {
		t.Fatalf("registry.NewClient: %v", err)
	}
	return c
}

func TestDownloadVerify(t *testing.T) {
	tarData := createTarContent(t)
	setupDir := t.TempDir()
	tarballPath := filepath.Join(setupDir, "test.tar.gz")
	compressGzip(t, tarData, tarballPath)

	tarballBytes, _ := os.ReadFile(tarballPath)
	hash := sha256.Sum256(tarballBytes)
	checksum := fmt.Sprintf("sha256:%x", hash)

	t.Run("success", func(t *testing.T) {
		files := &registryFiles{}
		server := startTestServer(t, files, tarballPath)
		defer server.Close()

		files.index = buildIndex()
		files.goFile = buildGoFile(server.URL, checksum, len(tarballBytes))
		files.cmakeFile = buildCmakeFile(server.URL, checksum, len(tarballBytes))

		tmpDir := t.TempDir()
		paths := env.NewPathsWithRoot(filepath.Join(tmpDir, "compendium"))
		paths.EnsureDirs()

		items := []InstallItem{
			{Name: "go", Version: "1.22.0", Kind: "languages", Platform: runtime.GOOS, Arch: runtime.GOARCH},
		}

		err := DownloadVerify(items, paths, newClientForTest(t, server.URL))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		extractedPath := filepath.Join(paths.LanguageDir("go", "1.22.0"), "hello.txt")
		data, err := os.ReadFile(extractedPath)
		if err != nil {
			t.Fatalf("expected extracted file: %v", err)
		}
		if string(data) != testContentInstaller {
			t.Errorf("expected '%s', got '%s'", testContentInstaller, data)
		}
	})

	t.Run("bad checksum", func(t *testing.T) {
		files := &registryFiles{}
		server := startTestServer(t, files, tarballPath)
		defer server.Close()

		badChecksum := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
		files.index = buildIndex()
		files.goFile = buildGoFile(server.URL, badChecksum, len(tarballBytes))
		files.cmakeFile = buildCmakeFile(server.URL, checksum, len(tarballBytes))

		tmpDir := t.TempDir()
		paths := env.NewPathsWithRoot(filepath.Join(tmpDir, "compendium"))
		paths.EnsureDirs()

		items := []InstallItem{
			{Name: "go", Version: "1.22.0", Kind: "languages", Platform: runtime.GOOS, Arch: runtime.GOARCH},
		}

		err := DownloadVerify(items, paths, newClientForTest(t, server.URL))
		if err == nil {
			t.Error("expected error for bad checksum")
		}
	})

	t.Run("registry lookup fails", func(t *testing.T) {
		files := &registryFiles{}
		server := startTestServer(t, files, tarballPath)
		defer server.Close()

		files.index = buildIndex()
		files.goFile = buildGoFile(server.URL, checksum, len(tarballBytes))
		files.cmakeFile = buildCmakeFile(server.URL, checksum, len(tarballBytes))

		tmpDir := t.TempDir()
		paths := env.NewPathsWithRoot(filepath.Join(tmpDir, "compendium"))
		paths.EnsureDirs()

		// rust isn't in the index — Lookup should fail before any sub-file fetch.
		items := []InstallItem{
			{Name: "rust", Version: "1.77.0", Kind: "languages", Platform: runtime.GOOS, Arch: runtime.GOARCH},
		}

		err := DownloadVerify(items, paths, newClientForTest(t, server.URL))
		if err == nil {
			t.Error("expected error for tool not in index")
		}
	})

	t.Run("bad artifact URL", func(t *testing.T) {
		files := &registryFiles{}
		server := startTestServer(t, files, tarballPath)
		defer server.Close()

		files.index = buildIndex()
		// Point the artifact URL at an unreachable host.
		files.goFile = fmt.Sprintf(`{
			"1.22.0": {
				"%s": {
					"%s": {"url":"http://localhost:99999/test.tar.gz","checksum":"%s","size":%d,"strip":0}
				}
			}
		}`, runtime.GOOS, runtime.GOARCH, checksum, len(tarballBytes))
		files.cmakeFile = buildCmakeFile(server.URL, checksum, len(tarballBytes))

		tmpDir := t.TempDir()
		paths := env.NewPathsWithRoot(filepath.Join(tmpDir, "compendium"))
		paths.EnsureDirs()

		items := []InstallItem{
			{Name: "go", Version: "1.22.0", Kind: "languages", Platform: runtime.GOOS, Arch: runtime.GOARCH},
		}

		err := DownloadVerify(items, paths, newClientForTest(t, server.URL))
		if err == nil {
			t.Error("expected error for unreachable artifact URL")
		}
	})

	t.Run("success with tool", func(t *testing.T) {
		files := &registryFiles{}
		server := startTestServer(t, files, tarballPath)
		defer server.Close()

		files.index = buildIndex()
		files.goFile = buildGoFile(server.URL, checksum, len(tarballBytes))
		files.cmakeFile = buildCmakeFile(server.URL, checksum, len(tarballBytes))

		tmpDir := t.TempDir()
		paths := env.NewPathsWithRoot(filepath.Join(tmpDir, "compendium"))
		paths.EnsureDirs()

		items := []InstallItem{
			{Name: "cmake", Version: "3.28.1", Kind: "tools", Platform: runtime.GOOS, Arch: runtime.GOARCH},
		}

		err := DownloadVerify(items, paths, newClientForTest(t, server.URL))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		extractedPath := filepath.Join(paths.ToolDir("cmake", "3.28.1"), "hello.txt")
		data, err := os.ReadFile(extractedPath)
		if err != nil {
			t.Fatalf("expected extracted file: %v", err)
		}
		if string(data) != testContentInstaller {
			t.Errorf("expected '%s', got '%s'", testContentInstaller, data)
		}
	})
}
