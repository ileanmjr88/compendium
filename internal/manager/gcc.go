package manager

import (
	"path/filepath"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
	"github.com/ileanmjr88/compendium/internal/ui"
)

type GCCManager struct {
	paths *env.Paths
}

func (m *GCCManager) Name() string {
	return "gcc"
}

func (m *GCCManager) EnvVars(version string, platform string) []EnvVar {
	langDir := m.paths.LanguageDir("gcc", version)

	if platform == "darwin" {
		ui.Print(ui.Warning, "GCC is not supported for macOS", "use clang instead")
		return nil
	}

	vars := ccCxxBaseEnvVars(langDir, "gcc", "g++", "gdb")

	// Linux library paths
	lib64 := filepath.Join(langDir, "lib64")
	vars = append(vars, EnvVar{Name: "LIBRARY_PATH", Value: filepath.Join(langDir, "lib64"), Action: "prepend"})
	vars = append(vars, EnvVar{Name: "LDFLAGS", Value: "-Wl,-rpath," + lib64, Action: "set"})

	return vars
}

func (m *GCCManager) ProjectEnvVars(projectName string, packages config.Packages) []EnvVar {
	if packages["vcpkg"] == "" {
		return nil
	}
	installedDir := filepath.Join(m.paths.ProjectDir(projectName), "vcpkg-installed")
	return []EnvVar{
		{Name: "VCPKG_INSTALLED_DIR", Value: installedDir, Action: "set"},
		{Name: "PKG_CONFIG_PATH", Value: filepath.Join(installedDir, "lib", "pkgconfig"), Action: "prepend"},
		{Name: "VCPKG_DISABLE_METRICS", Value: "1", Action: "set"},
	}
}
