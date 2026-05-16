package manager

import (
	"path/filepath"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
)

type PythonManager struct {
	paths *env.Paths
}

func (m *PythonManager) Name() string {
	return "python"
}

func (m *PythonManager) EnvVars(version string, platform string) []EnvVar {
	vars := []EnvVar{
		{Name: "PYTHONHOME", Value: "", Action: "unset"},
	}

	if platform != "darwin" {
		libDir := filepath.Join(m.paths.LanguageDir("python", version), "lib")
		vars = append(vars, EnvVar{Name: "LD_LIBRARY_PATH", Value: libDir, Action: "prepend"})
	}
	return vars
}

func (m *PythonManager) ProjectEnvVars(projectName string, packages config.Packages) []EnvVar {
	venvDir := filepath.Join(m.paths.ProjectDir(projectName), "venv")
	return []EnvVar{
		{Name: "VIRTUAL_ENV", Value: venvDir, Action: "set"},
		{Name: "UV_PROJECT_ENVIRONMENT", Value: venvDir, Action: "set"},
	}
}
