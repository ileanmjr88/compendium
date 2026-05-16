package manager

import (
	"os"
	"path/filepath"

	"github.com/ileanmjr88/compendium/internal/ui"
)

func ccCxxBaseEnvVars(langDir, ccBin, cxxBin, dbgBin string) []EnvVar {
	vars := []EnvVar{
		{Name: "CC", Value: filepath.Join(langDir, "bin", ccBin), Action: "set"},
		{Name: "CXX", Value: filepath.Join(langDir, "bin", cxxBin), Action: "set"},
		{Name: "COMPENDIUM_CC_CXX_DIR", Value: langDir, Action: "set"},
	}
	dbg := filepath.Join(langDir, "bin", dbgBin)
	if _, err := os.Stat(dbg); err == nil {
		vars = append(vars, EnvVar{Name: "COMPENDIUM_DEBUGGER", Value: dbg, Action: "set"})
	} else {
		ui.Print(ui.Warning, "Debugger not found at "+dbg, "")
	}
	return vars
}
