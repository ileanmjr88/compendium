package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
	"github.com/ileanmjr88/compendium/internal/installer"
	"github.com/ileanmjr88/compendium/internal/registry"
	"github.com/ileanmjr88/compendium/internal/ui"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install everything declared in compendium.toml",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Load config
		cfg, result, err := config.Load("compendium.toml")
		if err != nil {
			ui.Print(ui.Fail, "loading config", err.Error())
			os.Exit(1)
		}
		if result.HasErrors() {
			for _, e := range result.Errors {
				ui.Print(ui.Fail, e, "")
			}
			os.Exit(1)
		}
		for _, w := range result.Warnings {
			ui.Print(ui.Warning, w, "")
		}

		// 2. Set up paths
		paths, err := env.NewPaths()
		if err != nil {
			ui.Print(ui.Fail, "setting up paths", err.Error())
			os.Exit(1)
		}
		if err := paths.EnsureDirs(); err != nil {
			ui.Print(ui.Fail, "creating directories", err.Error())
			os.Exit(1)
		}

		// Resolve project name: explicit or cwd basename
		if cfg.Compendium.Name == "" {
			dir, _ := os.Getwd()
			cfg.Compendium.Name = filepath.Base(dir)
		}

		// 3. Resolve install items from config
		items := installer.Resolve(*cfg)

		indexURL := registry.ResolveSource("public")
		ui.Print(ui.Action, "fetching index", "public")
		client, err := registry.NewClient(indexURL)
		if err != nil {
			ui.Print(ui.Fail, "fetching index", err.Error())
			os.Exit(1)
		}
		ui.Print(ui.Success, "index ok", "")

		if err := installer.ValidateDeps(items, client, *cfg); err != nil {
			ui.Print(ui.Fail, "dependency check", err.Error())
			os.Exit(1)
		}

		// 4. Filter out already installed
		items = installer.FilterInstalled(items, paths)

		// 5. Download, verify, and extract (skip if nothing to install)
		if len(items) == 0 {
			ui.Print(ui.Success, "languages/tools already installed", "")
		} else {
			if err := installer.DownloadVerify(items, paths, client); err != nil {
				ui.Print(ui.Fail, "install failed", err.Error())
				os.Exit(1)
			}
		}

		// 6. Set up project dirs (venvs, etc.) — always runs so project-level
		// state (venv version, go workspace) reflects the current config even
		// when all languages/tools were already installed globally.
		if err := ensureProjectDirs(*cfg, paths); err != nil {
			ui.Print(ui.Fail, "creating project directories", err.Error())
			os.Exit(1)
		}

		ui.Print(ui.Success, "environment ready", "")
	},
}

func ensureProjectDirs(cfg config.Config, paths *env.Paths) error {
	projectDir := paths.ProjectDir(cfg.Compendium.Name)

	// Go workspace
	if cfg.Languages["go"] != "" {
		for _, sub := range []string{"go", "go/bin"} {
			dir := filepath.Join(projectDir, sub)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("creating go dir: %w", err)
			}
		}
	}

	// Python venv
	if cfg.Languages["python"] != "" {
		venvDir := filepath.Join(projectDir, "venv")
		desired := cfg.Languages["python"]

		needsCreate := true
		versionChanged := false
		if actual, err := readVenvVersion(filepath.Join(venvDir, "pyvenv.cfg")); err == nil {
			if strings.HasPrefix(actual, desired) {
				needsCreate = false
			} else {
				versionChanged = true
			}
		}

		if needsCreate {
			if versionChanged {
				ui.Print(ui.Warning, "python version changed", "venv recreated — reinstall packages")
			}
			_ = os.RemoveAll(venvDir)
			pythonBin := filepath.Join(paths.LanguageDir("python", desired), "bin", "python3")
			cmd := exec.Command(pythonBin, "-m", "venv", venvDir)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("creating python venv: %s: %w", string(out), err)
			}
		}
	}

	// vcpkg installed packages
	if cfg.Packages["vcpkg"] != "" {
		dir := filepath.Join(projectDir, "vcpkg-installed")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating vcpkg-installed dir: %w", err)
		}
	}

	return nil
}

func readVenvVersion(cfgPath string) (string, error) {
	f, err := os.Open(cfgPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(key) == "version" {
			return strings.TrimSpace(value), nil
		}
	}

	return "", fmt.Errorf("version not found in %s", cfgPath)
}

func init() {
	rootCmd.AddCommand(installCmd)
}
