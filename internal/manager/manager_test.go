package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
)

// stubDebugger creates an empty file at langDir/bin/<name> so os.Stat succeeds
// and COMPENDIUM_DEBUGGER gets emitted. Returns the root paths to pass to the manager.
func stubDebugger(t *testing.T, section, tool, version, dbgName string) *env.Paths {
	t.Helper()
	root := t.TempDir()
	paths := env.NewPathsWithRoot(root)
	binDir := filepath.Join(root, section, tool, version, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	f, err := os.Create(filepath.Join(binDir, dbgName))
	if err != nil {
		t.Fatalf("create stub: %v", err)
	}
	f.Close()
	return paths
}

func TestGCCManager_EnvVars_Linux(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &GCCManager{paths: paths}

	vars := mgr.EnvVars("13.2.0", "linux")

	if len(vars) != 5 {
		t.Fatalf("expected 5 env vars, got %d", len(vars))
	}

	lib64 := "/home/user/.local/compendium/languages/gcc/13.2.0/lib64"
	foundCC := false
	foundCXX := false
	foundLibraryPath := false
	foundLDFlags := false
	for _, v := range vars {
		switch v.Name {
		case "CC":
			foundCC = true
			if v.Value != "/home/user/.local/compendium/languages/gcc/13.2.0/bin/gcc" {
				t.Errorf("CC value: got %s", v.Value)
			}
		case "CXX":
			foundCXX = true
			if v.Value != "/home/user/.local/compendium/languages/gcc/13.2.0/bin/g++" {
				t.Errorf("CXX value: got %s", v.Value)
			}
		case "LIBRARY_PATH":
			foundLibraryPath = true
			if v.Value != lib64 {
				t.Errorf("LIBRARY_PATH value: got %s", v.Value)
			}
			if v.Action != "prepend" {
				t.Errorf("LIBRARY_PATH action: expected prepend, got %s", v.Action)
			}
		case "LDFLAGS":
			foundLDFlags = true
			if !strings.Contains(v.Value, "-Wl,-rpath,"+lib64) {
				t.Errorf("LDFLAGS missing rpath: got %s", v.Value)
			}
			if v.Action != "set" {
				t.Errorf("LDFLAGS action: expected set, got %s", v.Action)
			}
		case "LD_LIBRARY_PATH":
			t.Errorf("LD_LIBRARY_PATH should not be set (rpath goes in LDFLAGS): got %s", v.Value)
		}
	}
	if !foundCC {
		t.Error("missing CC")
	}
	if !foundCXX {
		t.Error("missing CXX")
	}
	if !foundLibraryPath {
		t.Error("missing LIBRARY_PATH for lib64")
	}
	if !foundLDFlags {
		t.Error("missing LDFLAGS with rpath")
	}
}

func TestGCCManager_EnvVars_IncludesCCCXXDir(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &GCCManager{paths: paths}
	vars := mgr.EnvVars("13.2.0", "linux")

	found := false
	for _, v := range vars {
		if v.Name == "COMPENDIUM_CC_CXX_DIR" {
			found = true
			if v.Value != "/home/user/.local/compendium/languages/gcc/13.2.0" {
				t.Errorf("COMPENDIUM_CC_CXX_DIR value: got %s", v.Value)
			}
		}
	}
	if !found {
		t.Error("missing COMPENDIUM_CC_CXX_DIR")
	}
}

func TestGCCManager_EnvVars_IncludesDebuggerWhenPresent(t *testing.T) {
	paths := stubDebugger(t, "languages", "gcc", "13.2.0", "gdb")
	mgr := &GCCManager{paths: paths}
	vars := mgr.EnvVars("13.2.0", "linux")

	for _, v := range vars {
		if v.Name == "COMPENDIUM_DEBUGGER" {
			if !strings.HasSuffix(v.Value, "/bin/gdb") {
				t.Errorf("COMPENDIUM_DEBUGGER value: expected gdb path, got %s", v.Value)
			}
			return
		}
	}
	t.Error("missing COMPENDIUM_DEBUGGER when gdb exists")
}

func TestGCCManager_EnvVars_Darwin(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &GCCManager{paths: paths}

	vars := mgr.EnvVars("13.2.0", "darwin")

	// GCC is not supported on macOS — should return nil
	if vars != nil {
		t.Errorf("expected nil env vars on darwin, got %d vars", len(vars))
	}
}

func TestClangManager_EnvVars_Linux(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &ClangManager{paths: paths}

	vars := mgr.EnvVars("22.1.2", "linux")

	if len(vars) != 6 {
		t.Fatalf("expected 6 env vars, got %d", len(vars))
	}

	foundCC := false
	foundCXX := false
	foundCXXFLAGS := false
	foundLDFLAGS := false
	foundLibraryPath := false
	for _, v := range vars {
		switch v.Name {
		case "CC":
			foundCC = true
			if v.Value != "/home/user/.local/compendium/languages/clang/22.1.2/bin/clang" {
				t.Errorf("CC value: got %s", v.Value)
			}
		case "CXX":
			foundCXX = true
			if v.Value != "/home/user/.local/compendium/languages/clang/22.1.2/bin/clang++" {
				t.Errorf("CXX value: got %s", v.Value)
			}
		case "CXXFLAGS":
			foundCXXFLAGS = true
			if v.Value != "-stdlib=libc++" {
				t.Errorf("CXXFLAGS value: expected -stdlib=libc++, got %s", v.Value)
			}
		case "LDFLAGS":
			foundLDFLAGS = true
			if !strings.Contains(v.Value, "-stdlib=libc++") ||
				!strings.Contains(v.Value, "-fuse-ld=lld") ||
				!strings.Contains(v.Value, "-Wl,-rpath,") {
				t.Errorf("LDFLAGS missing expected flags: got %s", v.Value)
			}
		case "LIBRARY_PATH":
			foundLibraryPath = true
			if v.Action != "prepend" {
				t.Errorf("LIBRARY_PATH action: expected prepend, got %s", v.Action)
			}
		case "LD_LIBRARY_PATH":
			t.Errorf("LD_LIBRARY_PATH should not be set (rpath goes in LDFLAGS): got %s", v.Value)
		}
	}
	if !foundCC {
		t.Error("missing CC")
	}
	if !foundCXX {
		t.Error("missing CXX")
	}
	if !foundCXXFLAGS {
		t.Error("missing CXXFLAGS with -stdlib=libc++")
	}
	if !foundLDFLAGS {
		t.Error("missing LDFLAGS with rpath")
	}
	if !foundLibraryPath {
		t.Error("missing LIBRARY_PATH")
	}
}

func TestClangManager_EnvVars_IncludesCCCXXDir(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &ClangManager{paths: paths}
	vars := mgr.EnvVars("22.1.2", "linux")

	found := false
	for _, v := range vars {
		if v.Name == "COMPENDIUM_CC_CXX_DIR" {
			found = true
			if v.Value != "/home/user/.local/compendium/languages/clang/22.1.2" {
				t.Errorf("COMPENDIUM_CC_CXX_DIR value: got %s", v.Value)
			}
		}
	}
	if !found {
		t.Error("missing COMPENDIUM_CC_CXX_DIR")
	}
}

func TestClangManager_EnvVars_IncludesDebuggerWhenPresent(t *testing.T) {
	paths := stubDebugger(t, "languages", "clang", "22.1.2", "lldb-dap")
	mgr := &ClangManager{paths: paths}
	vars := mgr.EnvVars("22.1.2", "linux")

	for _, v := range vars {
		if v.Name == "COMPENDIUM_DEBUGGER" {
			if !strings.HasSuffix(v.Value, "/bin/lldb-dap") {
				t.Errorf("COMPENDIUM_DEBUGGER value: expected lldb-dap path, got %s", v.Value)
			}
			return
		}
	}
	t.Error("missing COMPENDIUM_DEBUGGER when lldb-dap exists")
}

func TestArmNoneEabiGccManager_EnvVars(t *testing.T) {
	paths := stubDebugger(t, "languages", "arm-none-eabi-gcc", "14.2.0", "arm-none-eabi-gdb")
	mgr := &ArmNoneEabiGccManager{paths: paths}
	vars := mgr.EnvVars("14.2.0", "linux")

	want := map[string]string{
		"CC":                    "bin/arm-none-eabi-gcc",
		"CXX":                   "bin/arm-none-eabi-g++",
		"COMPENDIUM_CC_CXX_DIR": "arm-none-eabi-gcc/14.2.0",
		"COMPENDIUM_DEBUGGER":   "bin/arm-none-eabi-gdb",
	}
	found := map[string]bool{}
	for _, v := range vars {
		if suffix, ok := want[v.Name]; ok {
			if !strings.Contains(v.Value, suffix) {
				t.Errorf("%s value: expected suffix %s, got %s", v.Name, suffix, v.Value)
			}
			found[v.Name] = true
		}
	}
	for name := range want {
		if !found[name] {
			t.Errorf("missing %s", name)
		}
	}
}

func TestRiscvNoneElfGccManager_EnvVars(t *testing.T) {
	paths := stubDebugger(t, "languages", "riscv-none-elf-gcc", "14.2.0", "riscv-none-elf-gdb")
	mgr := &RiscvNoneElfGccManager{paths: paths}
	vars := mgr.EnvVars("14.2.0", "linux")

	want := map[string]string{
		"CC":                    "bin/riscv-none-elf-gcc",
		"CXX":                   "bin/riscv-none-elf-g++",
		"COMPENDIUM_CC_CXX_DIR": "riscv-none-elf-gcc/14.2.0",
		"COMPENDIUM_DEBUGGER":   "bin/riscv-none-elf-gdb",
	}
	found := map[string]bool{}
	for _, v := range vars {
		if suffix, ok := want[v.Name]; ok {
			if !strings.Contains(v.Value, suffix) {
				t.Errorf("%s value: expected suffix %s, got %s", v.Name, suffix, v.Value)
			}
			found[v.Name] = true
		}
	}
	for name := range want {
		if !found[name] {
			t.Errorf("missing %s", name)
		}
	}
}

func TestArmAndRiscvManagers_ProjectEnvVarsNil(t *testing.T) {
	paths := env.NewPathsWithRoot("/tmp/test")
	arm := &ArmNoneEabiGccManager{paths: paths}
	riscv := &RiscvNoneElfGccManager{paths: paths}
	if arm.ProjectEnvVars("myapp", config.Packages{"vcpkg": "vcpkg.json"}) != nil {
		t.Error("arm ProjectEnvVars should return nil")
	}
	if riscv.ProjectEnvVars("myapp", config.Packages{"vcpkg": "vcpkg.json"}) != nil {
		t.Error("riscv ProjectEnvVars should return nil")
	}
}

func TestClangManager_EnvVars_Darwin(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &ClangManager{paths: paths}

	vars := mgr.EnvVars("22.1.2", "darwin")

	// macOS: CC, CXX, SDKROOT, CXXFLAGS — no DYLD_LIBRARY_PATH, no LDFLAGS
	foundCXXFLAGS := false
	foundLDFLAGS := false
	foundDYLD := false
	foundSDKROOT := false
	for _, v := range vars {
		if v.Name == "CXXFLAGS" {
			foundCXXFLAGS = true
		}
		if v.Name == "LDFLAGS" {
			foundLDFLAGS = true
		}
		if v.Name == "DYLD_LIBRARY_PATH" {
			foundDYLD = true
		}
		if v.Name == "SDKROOT" {
			foundSDKROOT = true
		}
	}
	if !foundCXXFLAGS {
		t.Error("expected CXXFLAGS on darwin (with -nostdinc++)")
	}
	if foundLDFLAGS {
		t.Error("should not set LDFLAGS on darwin")
	}
	if foundDYLD {
		t.Error("should not set DYLD_LIBRARY_PATH on darwin")
	}
	if !foundSDKROOT {
		t.Error("expected SDKROOT on darwin")
	}
}

func TestGoManager_EnvVars(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &GoLangManager{paths: paths}

	vars := mgr.EnvVars("1.24.1", "linux")

	if len(vars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(vars))
	}

	for _, v := range vars {
		switch v.Name {
		case "GOROOT":
			if v.Value != "/home/user/.local/compendium/languages/go/1.24.1" {
				t.Errorf("GOROOT value: expected go lang dir, got %s", v.Value)
			}
			if v.Action != "set" {
				t.Errorf("GOROOT action: expected set, got %s", v.Action)
			}
		case "GOTOOLCHAIN":
			if v.Value != "local" {
				t.Errorf("GOTOOLCHAIN value: expected 'local', got %s", v.Value)
			}
		default:
			t.Errorf("unexpected env var: %s", v.Name)
		}
	}
}

func TestPythonManager_EnvVars_Linux(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &PythonManager{paths: paths}

	vars := mgr.EnvVars("3.12.0", "linux")

	if len(vars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(vars))
	}

	foundHome, foundLib := false, false
	wantLib := "/home/user/.local/compendium/languages/python/3.12.0/lib"
	for _, v := range vars {
		switch v.Name {
		case "PYTHONHOME":
			foundHome = true
			if v.Action != "unset" {
				t.Errorf("PYTHONHOME action: got %s, want unset", v.Action)
			}
		case "LD_LIBRARY_PATH":
			foundLib = true
			if v.Action != "prepend" {
				t.Errorf("LD_LIBRARY_PATH action: got %s, want prepend", v.Action)
			}
			if v.Value != wantLib {
				t.Errorf("LD_LIBRARY_PATH value: got %s, want %s", v.Value, wantLib)
			}
		}
	}
	if !foundHome {
		t.Error("missing PYTHONHOME")
	}
	if !foundLib {
		t.Error("missing LD_LIBRARY_PATH")
	}
}

func TestPythonManager_EnvVars_Darwin(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &PythonManager{paths: paths}

	vars := mgr.EnvVars("3.12.0", "darwin")

	if len(vars) != 1 {
		t.Fatalf("expected 1 env var on darwin, got %d", len(vars))
	}
	if vars[0].Name != "PYTHONHOME" {
		t.Errorf("expected PYTHONHOME, got %s", vars[0].Name)
	}
	for _, v := range vars {
		if v.Name == "LD_LIBRARY_PATH" || v.Name == "DYLD_LIBRARY_PATH" {
			t.Errorf("should not set %s on darwin", v.Name)
		}
	}
}

func TestNodeManager_EnvVars(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &NodeManager{paths: paths}

	vars := mgr.EnvVars("20.11.0", "linux")

	if vars != nil {
		t.Errorf("expected nil env vars for node, got %v", vars)
	}
}

func TestForLanguage_ReturnsCorrectManager(t *testing.T) {
	paths := env.NewPathsWithRoot("/tmp/test")

	tests := []struct {
		name     string
		expected string
	}{
		{"go", "go"},
		{"gcc", "gcc"},
		{"clang", "clang"},
		{"python", "python"},
		{"node", "node"},
		{"arm-none-eabi-gcc", "arm-none-eabi-gcc"},
		{"riscv-none-elf-gcc", "riscv-none-elf-gcc"},
	}

	for _, tt := range tests {
		mgr := ForLanguage(tt.name, paths)
		if mgr == nil {
			t.Errorf("ForLanguage(%q) returned nil", tt.name)
			continue
		}
		if mgr.Name() != tt.expected {
			t.Errorf("ForLanguage(%q).Name() = %q, want %q", tt.name, mgr.Name(), tt.expected)
		}
	}
}

func TestForLanguage_ReturnsNilForUnknown(t *testing.T) {
	paths := env.NewPathsWithRoot("/tmp/test")

	mgr := ForLanguage("cmake", paths)
	if mgr != nil {
		t.Errorf("expected nil for unknown language, got %v", mgr)
	}
}

func TestGoManager_ProjectEnvVars(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &GoLangManager{paths: paths}

	vars := mgr.ProjectEnvVars("myapp", config.Packages{})

	if len(vars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(vars))
	}
	for _, v := range vars {
		switch v.Name {
		case "GOPATH":
			if v.Value != "/home/user/.local/compendium/envs/myapp/go" {
				t.Errorf("GOPATH value: expected go dir, got %s", v.Value)
			}
			if v.Action != "set" {
				t.Errorf("GOPATH action: expected set, got %s", v.Action)
			}
		case "GOBIN":
			if v.Value != "/home/user/.local/compendium/envs/myapp/go/bin" {
				t.Errorf("GOBIN value: expected go/bin dir, got %s", v.Value)
			}
			if v.Action != "set" {
				t.Errorf("GOBIN action: expected set, got %s", v.Action)
			}
		default:
			t.Errorf("unexpected env var: %s", v.Name)
		}
	}
}

func TestPythonManager_ProjectEnvVars(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &PythonManager{paths: paths}

	vars := mgr.ProjectEnvVars("myapp", config.Packages{})

	if len(vars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(vars))
	}
	expectedVenv := "/home/user/.local/compendium/envs/myapp/venv"
	want := map[string]bool{"VIRTUAL_ENV": false, "UV_PROJECT_ENVIRONMENT": false}
	for _, v := range vars {
		if _, ok := want[v.Name]; !ok {
			t.Errorf("unexpected env var: %s", v.Name)
			continue
		}
		want[v.Name] = true
		if v.Value != expectedVenv {
			t.Errorf("%s value: got %s, want %s", v.Name, v.Value, expectedVenv)
		}
		if v.Action != "set" {
			t.Errorf("%s action: got %s, want set", v.Name, v.Action)
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing env var: %s", name)
		}
	}
}

func TestGCCManager_ProjectEnvVars_WithVcpkg(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &GCCManager{paths: paths}

	vars := mgr.ProjectEnvVars("myapp", config.Packages{"vcpkg": "vcpkg.json"})

	if len(vars) != 3 {
		t.Fatalf("expected 3 env vars, got %d", len(vars))
	}
	for _, v := range vars {
		switch v.Name {
		case "VCPKG_INSTALLED_DIR":
			if v.Value != "/home/user/.local/compendium/envs/myapp/vcpkg-installed" {
				t.Errorf("VCPKG_INSTALLED_DIR value: got %s", v.Value)
			}
			if v.Action != "set" {
				t.Errorf("expected set action, got %s", v.Action)
			}
		case "PKG_CONFIG_PATH":
			if v.Value != "/home/user/.local/compendium/envs/myapp/vcpkg-installed/lib/pkgconfig" {
				t.Errorf("PKG_CONFIG_PATH value: got %s", v.Value)
			}
			if v.Action != "prepend" {
				t.Errorf("expected prepend action, got %s", v.Action)
			}
		case "VCPKG_DISABLE_METRICS":
			if v.Value != "1" {
				t.Errorf("VCPKG_DISABLE_METRICS value: expected 1, got %s", v.Value)
			}
			if v.Action != "set" {
				t.Errorf("expected set action, got %s", v.Action)
			}
		default:
			t.Errorf("unexpected env var: %s", v.Name)
		}
	}
}

func TestGCCManager_ProjectEnvVars_Empty(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &GCCManager{paths: paths}

	vars := mgr.ProjectEnvVars("myapp", config.Packages{})

	if vars != nil {
		t.Errorf("expected nil, got %v", vars)
	}
}

func TestClangManager_ProjectEnvVars_WithVcpkg(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &ClangManager{paths: paths}

	vars := mgr.ProjectEnvVars("myapp", config.Packages{"vcpkg": "vcpkg.json"})

	if len(vars) != 3 {
		t.Fatalf("expected 3 env vars, got %d", len(vars))
	}
	for _, v := range vars {
		switch v.Name {
		case "VCPKG_INSTALLED_DIR":
			if v.Action != "set" {
				t.Errorf("expected set action, got %s", v.Action)
			}
		case "PKG_CONFIG_PATH":
			if v.Action != "prepend" {
				t.Errorf("expected prepend action, got %s", v.Action)
			}
		case "VCPKG_DISABLE_METRICS":
			if v.Value != "1" {
				t.Errorf("VCPKG_DISABLE_METRICS value: expected 1, got %s", v.Value)
			}
		default:
			t.Errorf("unexpected env var: %s", v.Name)
		}
	}
}

func TestNodeManager_ProjectEnvVars(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &NodeManager{paths: paths}

	vars := mgr.ProjectEnvVars("myapp", config.Packages{"npm": "package.json"})

	if vars != nil {
		t.Errorf("expected nil, got %v", vars)
	}
}

func TestClangLinuxTriple(t *testing.T) {
	if got := clangLinuxTriple("arm64"); got != "aarch64-unknown-linux-gnu" {
		t.Errorf("arm64: got %s", got)
	}
	if got := clangLinuxTriple("amd64"); got != "x86_64-unknown-linux-gnu" {
		t.Errorf("amd64: got %s", got)
	}
	if got := clangLinuxTriple("riscv64"); got != "x86_64-unknown-linux-gnu" {
		t.Errorf("unknown arch should default to x86_64, got %s", got)
	}
}

func TestClangManager_ProjectEnvVars_Empty(t *testing.T) {
	paths := env.NewPathsWithRoot("/home/user/.local/compendium")
	mgr := &ClangManager{paths: paths}

	vars := mgr.ProjectEnvVars("myapp", config.Packages{})

	if vars != nil {
		t.Errorf("expected nil when vcpkg not configured, got %v", vars)
	}
}
