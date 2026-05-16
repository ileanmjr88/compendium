package manager

import (
	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
)

type ArmNoneEabiGccManager struct {
	paths *env.Paths
}

func (m *ArmNoneEabiGccManager) Name() string {
	return "arm-none-eabi-gcc"
}

func (m *ArmNoneEabiGccManager) EnvVars(version string, platform string) []EnvVar {
	langDir := m.paths.LanguageDir("arm-none-eabi-gcc", version)
	return ccCxxBaseEnvVars(langDir, "arm-none-eabi-gcc", "arm-none-eabi-g++", "arm-none-eabi-gdb")
}

func (m *ArmNoneEabiGccManager) ProjectEnvVars(projectName string, packages config.Packages) []EnvVar {
	return nil
}
