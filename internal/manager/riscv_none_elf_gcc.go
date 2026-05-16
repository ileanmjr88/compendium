package manager

import (
	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
)

type RiscvNoneElfGccManager struct {
	paths *env.Paths
}

func (m *RiscvNoneElfGccManager) Name() string {
	return "riscv-none-elf-gcc"
}

func (m *RiscvNoneElfGccManager) EnvVars(version string, platform string) []EnvVar {
	langDir := m.paths.LanguageDir("riscv-none-elf-gcc", version)
	return ccCxxBaseEnvVars(langDir, "riscv-none-elf-gcc", "riscv-none-elf-g++", "riscv-none-elf-gdb")
}

func (m *RiscvNoneElfGccManager) ProjectEnvVars(projectName string, packages config.Packages) []EnvVar {
	return nil
}
