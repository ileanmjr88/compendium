package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
	"github.com/ileanmjr88/compendium/internal/manager"
)

func TestResolveEnv_ReturnsCorrectPaths(t *testing.T) {
	tmpDir := t.TempDir()
	paths := env.NewPathsWithRoot(tmpDir)

	// Create bin dirs on disk
	goBin := filepath.Join(paths.LanguageDir("go", "1.24.1"), "bin")
	cmakeBin := filepath.Join(paths.ToolDir("cmake", "3.28.0"), "bin")
	os.MkdirAll(goBin, 0755)
	os.MkdirAll(cmakeBin, 0755)

	cfg := config.Config{
		Languages: config.Languages{"go": "1.24.1"},
		Tools:     config.Tools{"cmake": "3.28.0"},
	}

	dirs, _, warnings := ResolveEnv(cfg, paths)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(dirs) != 2 {
		t.Fatalf("expected 2 dirs, got %d", len(dirs))
	}

	found := strings.Join(dirs, ":")
	if !strings.Contains(found, goBin) {
		t.Errorf("expected dirs to contain %s", goBin)
	}
	if !strings.Contains(found, cmakeBin) {
		t.Errorf("expected dirs to contain %s", cmakeBin)
	}
}

func TestResolveEnv_WarnsOnMissing(t *testing.T) {
	tmpDir := t.TempDir()
	paths := env.NewPathsWithRoot(tmpDir)

	// Don't create any dirs on disk
	cfg := config.Config{
		Languages: config.Languages{"go": "1.24.1"},
		Tools:     config.Tools{"cmake": "3.28.0"},
	}

	dirs, _, warnings := ResolveEnv(cfg, paths)

	if len(dirs) != 0 {
		t.Errorf("expected no dirs, got %v", dirs)
	}
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(warnings))
	}
}

func TestResolveEnv_HandlesAtSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	paths := env.NewPathsWithRoot(tmpDir)

	// c = "gcc@12.2.1" should resolve to languages/gcc/12.2.1/bin
	gccBin := filepath.Join(paths.LanguageDir("gcc", "12.2.1"), "bin")
	os.MkdirAll(gccBin, 0755)

	cfg := config.Config{
		Languages: config.Languages{"c": "gcc@12.2.1"},
	}

	dirs, _, warnings := ResolveEnv(cfg, paths)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(dirs) != 1 {
		t.Fatalf("expected 1 dir, got %d", len(dirs))
	}
	if dirs[0] != gccBin {
		t.Errorf("expected %s, got %s", gccBin, dirs[0])
	}
}

func TestResolveEnv_ReturnsEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	paths := env.NewPathsWithRoot(tmpDir)

	goBin := filepath.Join(paths.LanguageDir("go", "1.24.1"), "bin")
	os.MkdirAll(goBin, 0755)

	cfg := config.Config{
		Languages: config.Languages{"go": "1.24.1"},
	}

	_, envVars, _ := ResolveEnv(cfg, paths)

	if len(envVars) == 0 {
		t.Fatal("expected env vars for go, got none")
	}

	foundGOROOT := false
	foundGOTOOLCHAIN := false
	for _, ev := range envVars {
		if ev.Name == "GOROOT" {
			foundGOROOT = true
			if ev.Action != "set" {
				t.Errorf("expected GOROOT action 'set', got '%s'", ev.Action)
			}
		}
		if ev.Name == "GOTOOLCHAIN" {
			foundGOTOOLCHAIN = true
			if ev.Value != "local" {
				t.Errorf("expected GOTOOLCHAIN value 'local', got '%s'", ev.Value)
			}
		}
	}
	if !foundGOROOT {
		t.Error("expected GOROOT in env vars")
	}
	if !foundGOTOOLCHAIN {
		t.Error("expected GOTOOLCHAIN in env vars")
	}
}

func TestActivateScript_Format(t *testing.T) {
	dirs := []string{"/home/user/.local/compendium/languages/go/1.24.1/bin", "/home/user/.local/compendium/tools/cmake/3.28.0/bin"}
	script := ActivateScript(dirs, nil)

	if !strings.Contains(script, `_COMPENDIUM_OLD_PATH`) {
		t.Error("script should save old PATH")
	}
	if !strings.Contains(script, `export PATH="/home/user/.local/compendium/languages/go/1.24.1/bin:/home/user/.local/compendium/tools/cmake/3.28.0/bin:$PATH"`) {
		t.Error("script should export PATH with dirs prepended")
	}
	if !strings.Contains(script, `export _COMPENDIUM_ACTIVE=1`) {
		t.Error("script should set _COMPENDIUM_ACTIVE=1")
	}
}

func TestActivateScript_IdempotentOldPath(t *testing.T) {
	dirs := []string{"/some/bin"}
	script := ActivateScript(dirs, nil)

	// Should only save _COMPENDIUM_OLD_PATH if not already set
	if !strings.Contains(script, `if [ -z "$_COMPENDIUM_OLD_PATH" ]`) {
		t.Error("script should guard _COMPENDIUM_OLD_PATH with -z check")
	}
}

func TestActivateScript_WithEnvVars(t *testing.T) {
	dirs := []string{"/some/bin"}
	envVars := []manager.EnvVar{
		{Name: "GOROOT", Value: "/path/to/go", Action: "set"},
		{Name: "CC", Value: "/path/to/gcc", Action: "set"},
	}
	script := ActivateScript(dirs, envVars)

	if !strings.Contains(script, `export GOROOT="/path/to/go"`) {
		t.Error("script should export GOROOT")
	}
	if !strings.Contains(script, `export CC="/path/to/gcc"`) {
		t.Error("script should export CC")
	}
	if !strings.Contains(script, `_COMPENDIUM_ORIG_GOROOT`) {
		t.Error("script should save original GOROOT")
	}
	if !strings.Contains(script, `_COMPENDIUM_DEACTIVATE`) {
		t.Error("script should store deactivate commands")
	}
}

func TestActivateScript_WithUnsetAction(t *testing.T) {
	dirs := []string{"/some/bin"}
	envVars := []manager.EnvVar{
		{Name: "PYTHONHOME", Value: "", Action: "unset"},
	}
	script := ActivateScript(dirs, envVars)

	if !strings.Contains(script, `unset PYTHONHOME`) {
		t.Error("script should unset PYTHONHOME")
	}
	if !strings.Contains(script, `_COMPENDIUM_ORIG_PYTHONHOME`) {
		t.Error("script should save original PYTHONHOME")
	}
}

func TestActivateScript_WithPrependAction(t *testing.T) {
	dirs := []string{"/some/bin"}
	envVars := []manager.EnvVar{
		{Name: "LD_LIBRARY_PATH", Value: "/path/to/libexec", Action: "prepend"},
	}
	script := ActivateScript(dirs, envVars)

	if !strings.Contains(script, `export LD_LIBRARY_PATH="/path/to/libexec:$LD_LIBRARY_PATH"`) {
		t.Error("script should prepend to LD_LIBRARY_PATH when set")
	}
	if !strings.Contains(script, `export LD_LIBRARY_PATH="/path/to/libexec"`) {
		t.Error("script should set LD_LIBRARY_PATH without colon when empty")
	}
}

func TestDeactivateScript_Format(t *testing.T) {
	script := DeactivateScript()

	if !strings.Contains(script, `export PATH="$_COMPENDIUM_OLD_PATH"`) {
		t.Error("script should restore PATH from _COMPENDIUM_OLD_PATH")
	}
	if !strings.Contains(script, `unset _COMPENDIUM_OLD_PATH`) {
		t.Error("script should unset _COMPENDIUM_OLD_PATH")
	}
	if !strings.Contains(script, `unset _COMPENDIUM_ACTIVE`) {
		t.Error("script should unset _COMPENDIUM_ACTIVE")
	}
}

func TestDeactivateScript_GuardsEmptyPath(t *testing.T) {
	script := DeactivateScript()

	if !strings.Contains(script, `if [ -n "$_COMPENDIUM_OLD_PATH" ]`) {
		t.Error("script should guard against restoring empty PATH")
	}
}

func TestDeactivateScript_EvalsDeactivateCommands(t *testing.T) {
	script := DeactivateScript()

	if !strings.Contains(script, `eval "$_COMPENDIUM_DEACTIVATE"`) {
		t.Error("script should eval _COMPENDIUM_DEACTIVATE commands")
	}
	if !strings.Contains(script, `unset _COMPENDIUM_DEACTIVATE`) {
		t.Error("script should unset _COMPENDIUM_DEACTIVATE")
	}
}

func TestActivateScript_SetsPromptIndicator(t *testing.T) {
	dirs := []string{"/some/bin"}
	script := ActivateScript(dirs, nil)

	if !strings.Contains(script, `_COMPENDIUM_OLD_PS1`) {
		t.Error("script should save old PS1")
	}
	if !strings.Contains(script, `(compendium) $PS1`) {
		t.Error("script should prepend (compendium) to PS1")
	}
}

func TestDeactivateScript_RestoresPrompt(t *testing.T) {
	script := DeactivateScript()

	if !strings.Contains(script, `export PS1="$_COMPENDIUM_OLD_PS1"`) {
		t.Error("script should restore PS1 from _COMPENDIUM_OLD_PS1")
	}
	if !strings.Contains(script, `unset _COMPENDIUM_OLD_PS1`) {
		t.Error("script should unset _COMPENDIUM_OLD_PS1")
	}
}

func TestResolveEnv_ProjectEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	paths := env.NewPathsWithRoot(tmpDir)

	// Create bin dirs on disk
	goBin := filepath.Join(paths.LanguageDir("go", "1.24.1"), "bin")
	pythonBin := filepath.Join(paths.LanguageDir("python", "3.12"), "bin")
	os.MkdirAll(goBin, 0755)
	os.MkdirAll(pythonBin, 0755)

	cfg := config.Config{
		Compendium: config.Compendium{Name: "myapp"},
		Languages:  config.Languages{"go": "1.24.1", "python": "3.12"},
		Packages:   config.Packages{"pip": "requirements.txt"},
	}

	dirs, envVars, _ := ResolveEnv(cfg, paths)

	// Check GOBIN is in dirs
	expectedGoBin := filepath.Join(paths.ProjectDir("myapp"), "go", "bin")
	foundGoBinDir := false
	for _, d := range dirs {
		if d == expectedGoBin {
			foundGoBinDir = true
		}
	}
	if !foundGoBinDir {
		t.Error("expected GOBIN dir in dirs list")
	}

	// Check project env vars are present
	foundGOPATH := false
	foundGOBIN := false
	foundVIRTUALENV := false
	for _, ev := range envVars {
		switch ev.Name {
		case "GOPATH":
			foundGOPATH = true
		case "GOBIN":
			foundGOBIN = true
		case "VIRTUAL_ENV":
			foundVIRTUALENV = true
		}
	}
	if !foundGOPATH {
		t.Error("expected GOPATH in env vars")
	}
	if !foundGOBIN {
		t.Error("expected GOBIN in env vars")
	}
	if !foundVIRTUALENV {
		t.Error("expected VIRTUAL_ENV in env vars")
	}
}

func TestResolveEnv_NoBinFallback(t *testing.T) {
	tmpDir := t.TempDir()
	paths := env.NewPathsWithRoot(tmpDir)

	// Create tool dir WITHOUT bin/ subdirectory (like uv)
	toolDir := paths.ToolDir("uv", "0.11.3")
	os.MkdirAll(toolDir, 0755)

	cfg := config.Config{
		Tools: config.Tools{"uv": "0.11.3"},
	}

	dirs, _, warnings := ResolveEnv(cfg, paths)

	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
	if len(dirs) != 1 {
		t.Fatalf("expected 1 dir, got %d", len(dirs))
	}
	if dirs[0] != toolDir {
		t.Errorf("expected fallback to %s, got %s", toolDir, dirs[0])
	}
}
