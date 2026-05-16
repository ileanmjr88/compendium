package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPathsWithRoot(t *testing.T) {
	p := NewPathsWithRoot("/tmp/test")

	if p.Root != "/tmp/test" {
		t.Errorf("Root = %q, want %q", p.Root, "/tmp/test")
	}
	if p.Languages != "/tmp/test/languages" {
		t.Errorf("Languages = %q, want %q", p.Languages, "/tmp/test/languages")
	}
	if p.Tools != "/tmp/test/tools" {
		t.Errorf("Tools = %q, want %q", p.Tools, "/tmp/test/tools")
	}
	if p.Envs != "/tmp/test/envs" {
		t.Errorf("Envs = %q, want %q", p.Envs, "/tmp/test/envs")
	}
}

func TestLanguageDir(t *testing.T) {
	p := NewPathsWithRoot("/tmp/test")
	got := p.LanguageDir("python", "3.12")
	want := "/tmp/test/languages/python/3.12"
	if got != want {
		t.Errorf("LanguageDir = %q, want %q", got, want)
	}
}

func TestToolDir(t *testing.T) {
	p := NewPathsWithRoot("/tmp/test")
	got := p.ToolDir("gcc", "14.1")
	want := "/tmp/test/tools/gcc/14.1"
	if got != want {
		t.Errorf("ToolDir = %q, want %q", got, want)
	}
}

func TestEnvDir(t *testing.T) {
	p := NewPathsWithRoot("/tmp/test")
	got := p.EnvDir("myproject", "1.0.0")
	want := "/tmp/test/envs/myproject/1.0.0"
	if got != want {
		t.Errorf("EnvDir = %q, want %q", got, want)
	}
}

func TestProjectDir(t *testing.T) {
	p := NewPathsWithRoot("/tmp/test")
	got := p.ProjectDir("myapp")
	want := "/tmp/test/envs/myapp"
	if got != want {
		t.Errorf("ProjectDir = %q, want %q", got, want)
	}
}

func TestEnsureDirs(t *testing.T) {
	tmpDir := t.TempDir()
	p := NewPathsWithRoot(tmpDir)

	if err := p.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error: %v", err)
	}

	for _, dir := range []string{p.Root, p.Languages, p.Tools, p.Envs} {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("directory %q was not created", dir)
		}
	}
}

func TestEnsureDirsError(t *testing.T) {
	p := NewPathsWithRoot("/dev/null/invalid")

	err := p.EnsureDirs()
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestCheckDir(t *testing.T) {
	tmpDir := t.TempDir()
	paths := NewPathsWithRoot(tmpDir)
	paths.EnsureDirs()

	// doesn't exist yet
	if paths.CheckDir("languages", "go", "1.22.0") {
		t.Error("expected false for non-existent dir")
	}

	// create it
	os.MkdirAll(paths.LanguageDir("go", "1.22.0"), 0755)

	// now it exists
	if !paths.CheckDir("languages", "go", "1.22.0") {
		t.Error("expected true for existing dir")
	}

	// test tools kind
	if paths.CheckDir("tools", "cmake", "3.28.1") {
		t.Error("expected false for non-existent tool dir")
	}

	os.MkdirAll(paths.ToolDir("cmake", "3.28.1"), 0755)

	if !paths.CheckDir("tools", "cmake", "3.28.1") {
		t.Error("expected true for existing tool dir")
	}
}

func TestNewPaths(t *testing.T) {
	p, err := NewPaths()
	if err != nil {
		t.Fatalf("NewPaths() error: %v", err)
	}

	home, _ := os.UserHomeDir()
	expectedRoot := filepath.Join(home, ".local", "compendium")
	if p.Root != expectedRoot {
		t.Errorf("Root = %q, want %q", p.Root, expectedRoot)
	}
}
