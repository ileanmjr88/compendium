package installer

import (
	"os"
	"testing"

	"github.com/ileanmjr88/compendium/internal/env"
)

func TestFilterInstalled(t *testing.T) {
	t.Run("all new", func(t *testing.T) {
		tmpDir := t.TempDir()
		paths := env.NewPathsWithRoot(tmpDir)
		paths.EnsureDirs()

		items := []InstallItem{
			{Name: "go", Version: "1.22.0", Kind: "languages"},
		}

		result := FilterInstalled(items, paths)

		if len(result) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result))
		}
	})

	t.Run("some installed", func(t *testing.T) {
		tmpDir := t.TempDir()
		paths := env.NewPathsWithRoot(tmpDir)
		paths.EnsureDirs()

		os.MkdirAll(paths.LanguageDir("go", "1.22.0"), 0755)

		items := []InstallItem{
			{Name: "go", Version: "1.22.0", Kind: "languages"},
			{Name: "cmake", Version: "3.28.1", Kind: "tools"},
		}

		result := FilterInstalled(items, paths)

		if len(result) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result))
		}

		if result[0].Name != "cmake" {
			t.Errorf("expected name 'cmake', got '%s'", result[0].Name)
		}
	})

	t.Run("all installed", func(t *testing.T) {
		tmpDir := t.TempDir()
		paths := env.NewPathsWithRoot(tmpDir)
		paths.EnsureDirs()

		os.MkdirAll(paths.LanguageDir("go", "1.22.0"), 0755)
		os.MkdirAll(paths.ToolDir("cmake", "3.28.1"), 0755)

		items := []InstallItem{
			{Name: "go", Version: "1.22.0", Kind: "languages"},
			{Name: "cmake", Version: "3.28.1", Kind: "tools"},
		}

		result := FilterInstalled(items, paths)

		if len(result) != 0 {
			t.Errorf("expected 0 items, got %d", len(result))
		}
	})

	t.Run("empty items", func(t *testing.T) {
		tmpDir := t.TempDir()
		paths := env.NewPathsWithRoot(tmpDir)
		paths.EnsureDirs()

		items := []InstallItem{}

		result := FilterInstalled(items, paths)

		if len(result) != 0 {
			t.Errorf("expected 0 items, got %d", len(result))
		}
	})
}
