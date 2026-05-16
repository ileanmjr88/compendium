package manager

import (
	"github.com/ileanmjr88/compendium/internal/env"
)

func ForLanguage(name string, paths *env.Paths) LanguageManager {
	switch name {
	case "go":
		return &GoLangManager{paths: paths}
	case "gcc":
		return &GCCManager{paths: paths}
	case "clang":
		return &ClangManager{paths: paths}
	case "python":
		return &PythonManager{paths: paths}
	case "node":
		return &NodeManager{paths: paths}
	case "arm-none-eabi-gcc":
		return &ArmNoneEabiGccManager{paths: paths}
	case "riscv-none-elf-gcc":
		return &RiscvNoneElfGccManager{paths: paths}
	default:
		return nil
	}
}
