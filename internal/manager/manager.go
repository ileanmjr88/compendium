package manager

import "github.com/ileanmjr88/compendium/internal/config"

type EnvVar struct {
	Name   string // GOROOT, CC, LD_LIBRARY_PATH
	Value  string // value to be set (empty for unset)
	Action string // set, unset, prepend
}

type LanguageManager interface {
	Name() string
	EnvVars(version string, platform string) []EnvVar
	ProjectEnvVars(projectName string, packages config.Packages) []EnvVar
}
