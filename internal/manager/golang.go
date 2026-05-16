package manager

import (
	"path/filepath"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
)

type GoLangManager struct {
	paths *env.Paths
}

func (m *GoLangManager) Name() string {
	return "go"
}

func (m *GoLangManager) EnvVars(version string, platform string) []EnvVar {
	vars := []EnvVar{
		{Name: "GOROOT", Value: m.paths.LanguageDir("go", version), Action: "set"},
		{Name: "GOTOOLCHAIN", Value: "local", Action: "set"},
	}
	return vars
}

func (m *GoLangManager) ProjectEnvVars(projectName string, packages config.Packages) []EnvVar {
	goDir := filepath.Join(m.paths.ProjectDir(projectName), "go")
	return []EnvVar{
		{Name: "GOPATH", Value: goDir, Action: "set"},
		{Name: "GOBIN", Value: filepath.Join(goDir, "bin"), Action: "set"},
	}
}
