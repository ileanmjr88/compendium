package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	toml := `
[compendium]
name = "test-project"
version = "1.0.0"
min_compendium = "0.1.0"

[registry]
source = "public"

[languages]
node = "20.11.0"
python = "3.12"

[tools]
gcc = "13.2"
cmake = "3.28"

[target]
platform = "linux"
arch = "x86_64"

[packages]
npm = "package.json"
pip = "requirements.txt"

[hooks]
pre-commit = ["lint", "format --check"]


[scripts]
test = "pytest tests/"
`
	path := writeTempFile(t, toml)
	cfg, result, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if cfg.Compendium.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", cfg.Compendium.Version)
	}
	if cfg.Languages["node"] != "20.11.0" {
		t.Errorf("expected node 20.11.0, got %s", cfg.Languages["node"])
	}
	if cfg.Tools["gcc"] != "13.2" {
		t.Errorf("expected gcc 13.2, got %s", cfg.Tools["gcc"])
	}
	if cfg.Target.Platform != "linux" {
		t.Errorf("expected platform linux, got %s", cfg.Target.Platform)
	}
	if cfg.Target.Arch != "x86_64" {
		t.Errorf("expected arch x86_64, got %s", cfg.Target.Arch)
	}
	if cfg.Packages["npm"] != "package.json" {
		t.Errorf("expected package.json, got %s", cfg.Packages["npm"])
	}
	if cfg.Scripts["test"] != "pytest tests/" {
		t.Errorf("expected script, got %s", cfg.Scripts["test"])
	}
}

func TestLoadMissingFields(t *testing.T) {
	path := writeTempFile(t, "")
	_, result, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasErrors() {
		t.Error("expected validation errors from empty config")
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	path := writeTempFile(t, "this is not [valid toml ={{{")
	_, _, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, _, err := Load("/nonexistent/path.toml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestValidateNonPublicSource(t *testing.T) {
	toml := `
[compendium]
version = "1.0.0"
min_compendium = "0.1.0"

[registry]
source = "https://example.com/index.json"

`
	path := writeTempFile(t, toml)
	_, result, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasErrors() {
		t.Error(`expected error for non-"public" registry.source`)
	}
}

func TestValidateEmbeddedTarget(t *testing.T) {
	toml := `
[compendium]
version = "1.0.0"
min_compendium = "0.1.0"

[registry]
source = "public"

[target]
platform = "linux"
arch = "x86_64"
fpu = "fpv4-sp-d16"

`
	path := writeTempFile(t, toml)
	_, result, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasWarnings() {
		t.Error("expected warning for fpu on non-embedded platform")
	}
}

func TestValidateUnknownPlatform(t *testing.T) {
	toml := `
[compendium]
version = "1.0.0"
min_compendium = "0.1.0"

[registry]
source = "public"

[target]
platform = "freebsd"

`
	path := writeTempFile(t, toml)
	_, result, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasWarnings() {
		t.Error("expected warning for unknown platform 'freebsd'")
	}
}

func TestValidateUnknownArch(t *testing.T) {
	toml := `
[compendium]
version = "1.0.0"
min_compendium = "0.1.0"

[registry]
source = "public"

[target]
arch = "mips64"

`
	path := writeTempFile(t, toml)
	_, result, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasWarnings() {
		t.Error("expected warning for unknown architecture 'mips64'")
	}
}

// Helper function
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "compendium.toml")
	if err := os.WriteFile(path, []byte(content), 0664); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return path
}
