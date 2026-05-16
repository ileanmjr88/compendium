package manager

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
)

type ClangManager struct {
	paths *env.Paths
}

func (m *ClangManager) Name() string {
	return "clang"
}

func (m *ClangManager) EnvVars(version string, platform string) []EnvVar {
	langDir := m.paths.LanguageDir("clang", version)
	vars := ccCxxBaseEnvVars(langDir, "clang", "clang++", "lldb-dap")

	if platform == "darwin" {
		sdkPath, _ := exec.Command("xcrun", "--show-sdk-path").Output()
		sdk := strings.TrimSpace(string(sdkPath))
		vars = append(vars,
			EnvVar{Name: "SDKROOT", Value: sdk, Action: "set"},
			EnvVar{Name: "CXXFLAGS", Value: "-nostdinc++ -isystem " + sdk + "/usr/include/c++/v1", Action: "set"},
		)
	} else {
		triple := clangLinuxTriple(runtime.GOARCH)
		libDir := filepath.Join(langDir, "lib", triple)
		vars = append(vars, EnvVar{Name: "CXXFLAGS", Value: "-stdlib=libc++", Action: "set"})
		vars = append(vars, EnvVar{Name: "LDFLAGS", Value: "-stdlib=libc++ -fuse-ld=lld -Wl,-rpath," + libDir, Action: "set"})
		vars = append(vars, EnvVar{Name: "LIBRARY_PATH", Value: libDir, Action: "prepend"})
	}

	return vars
}

func (m *ClangManager) ProjectEnvVars(projectName string, packages config.Packages) []EnvVar {
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

func clangLinuxTriple(arch string) string {
	if arch == "arm64" {
		return "aarch64-unknown-linux-gnu"
	}
	return "x86_64-unknown-linux-gnu"
}
