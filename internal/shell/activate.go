package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
	"github.com/ileanmjr88/compendium/internal/installer"
	"github.com/ileanmjr88/compendium/internal/manager"
)

func ResolveEnv(cfg config.Config, paths *env.Paths) ([]string, []manager.EnvVar, []string) {
	items := installer.Resolve(cfg)
	var dirs []string
	var envVars []manager.EnvVar
	var warnings []string

	for _, item := range items {
		var baseDir string
		if item.Kind == "languages" {
			baseDir = paths.LanguageDir(item.Name, item.Version)
		} else {
			baseDir = paths.ToolDir(item.Name, item.Version)
		}

		binDir := filepath.Join(baseDir, "bin")
		if _, err := os.Stat(binDir); err == nil {
			dirs = append(dirs, binDir)
		} else if _, err := os.Stat(baseDir); err == nil {
			dirs = append(dirs, baseDir)
		} else {
			warnings = append(warnings, item.Name+" "+item.Version+": not installed")
		}

		mgr := manager.ForLanguage(item.Name, paths)
		if mgr != nil {
			envVars = append(envVars, mgr.EnvVars(item.Version, runtime.GOOS)...)
		}
	}

	if cfg.Compendium.Name != "" {
		seen := map[string]bool{}
		for _, item := range items {
			if seen[item.Name] {
				continue
			}
			seen[item.Name] = true
			mgr := manager.ForLanguage(item.Name, paths)
			if mgr != nil {
				envVars = append(envVars, mgr.ProjectEnvVars(cfg.Compendium.Name, cfg.Packages)...)
			}
		}
		if cfg.Languages["go"] != "" {
			goBin := filepath.Join(paths.ProjectDir(cfg.Compendium.Name), "go", "bin")
			dirs = append(dirs, goBin)
		}
		if cfg.Languages["python"] != "" {
			venvDir := filepath.Join(paths.ProjectDir(cfg.Compendium.Name), "venv")
			dirs = append([]string{filepath.Join(venvDir, "bin")}, dirs...)
		}
		if cfg.Tools["vcpkg"] != "" {
			vcpkgRoot := paths.ToolDir("vcpkg", cfg.Tools["vcpkg"])
			envVars = append(envVars, manager.EnvVar{Name: "VCPKG_ROOT", Value: vcpkgRoot, Action: "set"})
		}
	}
	return dirs, envVars, warnings
}

func ActivateScript(dirs []string, envVars []manager.EnvVar) string {
	var b strings.Builder

	// Bail early if already active — prevents PATH/LD_LIBRARY_PATH from doubling
	// on repeated `source <(compendium activate)`. Run `compendium deactivate`
	// first to re-activate with updated config.
	b.WriteString(`if [ -n "$_COMPENDIUM_ACTIVE" ]; then
  return 0 2>/dev/null || exit 0
fi
`)

	// PATH management
	path := strings.Join(dirs, ":")
	fmt.Fprintf(&b, `
if [ -z "$_COMPENDIUM_OLD_PATH" ]; then
  export _COMPENDIUM_OLD_PATH="$PATH"
fi
export _COMPENDIUM_PATHS="%s"
export PATH="%s:$PATH"
`, path, path)

	// Per-language env vars
	var varNames []string
	for _, ev := range envVars {
		origVar := "_COMPENDIUM_ORIG_" + ev.Name
		varNames = append(varNames, ev.Name)

		// Save original (idempotent — only if not already saved)
		fmt.Fprintf(&b, `if [ -z "$%s" ]; then
  if [ -n "$%s" ]; then
    export %s="$%s"
  else
    export %s="__COMPENDIUM_UNSET__"
  fi
fi
`, origVar, ev.Name, origVar, ev.Name, origVar)

		// Apply the action
		switch ev.Action {
		case "set":
			fmt.Fprintf(&b, `export %s="%s"
`, ev.Name, ev.Value)
		case "unset":
			fmt.Fprintf(&b, `unset %s
`, ev.Name)
		case "prepend":
			fmt.Fprintf(&b, `if [ -n "$%s" ]; then export %s="%s:$%s"; else export %s="%s"; fi
`, ev.Name, ev.Name, ev.Value, ev.Name, ev.Name, ev.Value)
		}
	}

	// Generate deactivate commands for each var and store in _COMPENDIUM_DEACTIVATE
	if len(varNames) > 0 {
		var deactivate strings.Builder
		for _, name := range varNames {
			origVar := "_COMPENDIUM_ORIG_" + name
			fmt.Fprintf(&deactivate,
				`if [ "$%s" = "__COMPENDIUM_UNSET__" ]; then unset %s; else export %s="$%s"; fi; unset %s; `,
				origVar, name, name, origVar, origVar)
		}
		fmt.Fprintf(&b, "export _COMPENDIUM_DEACTIVATE='%s'\n", deactivate.String())
	}

	// Shell prompt indicator
	b.WriteString(`if [ -z "$_COMPENDIUM_OLD_PS1" ]; then
  export _COMPENDIUM_OLD_PS1="$PS1"
  export PS1="(compendium) $PS1"
fi
`)

	b.WriteString("export _COMPENDIUM_ACTIVE=1\n")

	return b.String()
}

func DeactivateScript() string {
	var b strings.Builder

	// Restore PATH
	b.WriteString(`
if [ -n "$_COMPENDIUM_OLD_PATH" ]; then
  export PATH="$_COMPENDIUM_OLD_PATH"
fi
unset _COMPENDIUM_OLD_PATH
unset _COMPENDIUM_PATHS
`)

	// Restore per-language env vars (commands stored at activate time)
	b.WriteString(`if [ -n "$_COMPENDIUM_DEACTIVATE" ]; then
  eval "$_COMPENDIUM_DEACTIVATE"
fi
unset _COMPENDIUM_DEACTIVATE
if [ -n "$_COMPENDIUM_OLD_PS1" ]; then
  export PS1="$_COMPENDIUM_OLD_PS1"
fi
unset _COMPENDIUM_OLD_PS1
unset _COMPENDIUM_ACTIVE
`)

	return b.String()
}
