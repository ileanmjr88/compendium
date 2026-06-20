package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/ileanmjr88/compendium/internal/buildinfo"
	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
	"github.com/ileanmjr88/compendium/internal/installer"
	"github.com/ileanmjr88/compendium/internal/lockfile"
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

		// 2. Load the lock (empty on a first run, fail if corrupt) and diff the
		// toml against it to see what changed.
		lock := &lockfile.Lockfile{}
		lockExisted := false
		lfLoaded, err := lockfile.Load("compendium.lock")
		switch {
		case err == nil:
			lock = lfLoaded
			lockExisted = true
		case errors.Is(err, os.ErrNotExist):
			// no lock file → first run, keep the empty lock
		default:
			ui.Print(ui.Fail, "loading lockfile", err.Error())
			os.Exit(1)
		}

		changes := lockfile.Diff(*cfg, lock, runtime.GOOS, runtime.GOARCH)
		if frozen && len(changes) > 0 {
			ui.Print(ui.Fail, "compendium.lock is out of date", "run `compendium install` to update it")
			os.Exit(1)
		}

		// 3. Set up paths
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

		// 4. Resolve install items from config
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

		// 5. Filter out already installed
		items = installer.FilterInstalled(items, paths)

		// 6. Download, verify, and extract (skip if nothing to install)
		if len(items) == 0 {
			ui.Print(ui.Success, "languages/tools already installed", "")
		} else {
			if err := installer.DownloadVerify(items, paths, client); err != nil {
				ui.Print(ui.Fail, "install failed", err.Error())
				os.Exit(1)
			}
		}

		// 7. Set up project dirs (venvs, etc.) — always runs so project-level
		// state (venv version, go workspace) reflects the current config even
		// when all languages/tools were already installed globally.
		if err := ensureProjectDirs(*cfg, paths); err != nil {
			ui.Print(ui.Fail, "creating project directories", err.Error())
			os.Exit(1)
		}

		// 8. Update the lock only when something changed. A true no-op leaves the
		// file untouched: no rewrite, no meta drift.
		if len(changes) == 0 && lockExisted {
			ui.Print(ui.Success, "lockfile up to date", "")
		} else {
			resolved, err := buildResolved(changes, client)
			if err != nil {
				ui.Print(ui.Fail, "resolving lockfile entries", err.Error())
				os.Exit(1)
			}
			lock.Apply(changes, resolved)
			// Rebuild meta (schema/project/version/registry) but preserve the
			// grow-only platform set. buildMeta seeds only the current platform;
			// folding it into what the lock already carried — from Apply and from
			// other machines — keeps us from dropping platforms on every install.
			meta := buildMeta(*cfg, client)
			meta.Platforms = unionPlatforms(lock.Meta.Platforms, meta.Platforms)
			lock.Meta = meta
			if err := lockfile.Write("compendium.lock", lock); err != nil {
				ui.Print(ui.Fail, "writing lockfile", err.Error())
				os.Exit(1)
			}
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

// buildResolved looks up the registry artifact for every Added/SpecChanged
// change and converts it into a lockfile.Entry, grouped by section so
// (*Lockfile).Apply can upsert each into the right slice. Removed changes need
// no lookup. MVP records only the current platform's artifact; cross-platform
// population is a follow-up.
func buildResolved(changes []lockfile.Change, client *registry.Client) (lockfile.Resolved, error) {
	res := lockfile.Resolved{
		Languages: map[string]lockfile.Entry{},
		Tools:     map[string]lockfile.Entry{},
		Packages:  map[string]lockfile.EcoLockRef{},
	}
	platform, arch := runtime.GOOS, runtime.GOARCH

	for _, c := range changes {
		if c.Kind == lockfile.Removed {
			continue // nothing to resolve for a removal
		}

		switch c.Section {
		case lockfile.SectionLanguage, lockfile.SectionTool:
			// Look up by the registry name/version parsed from the spec
			// ("gcc@12.2.1" -> gcc, 12.2.1), but keep the config key as the
			// entry Name and the raw value as Spec so Diff stays stable.
			name, version := c.Name, c.NewSpec
			if strings.Contains(c.NewSpec, "@") {
				parts := strings.SplitN(c.NewSpec, "@", 2)
				name, version = parts[0], parts[1]
			}

			kind := "languages"
			if c.Section == lockfile.SectionTool {
				kind = "tools"
			}

			art, _, err := client.Lookup(kind, name, version, platform, arch)
			if err != nil {
				return res, fmt.Errorf("looking up %s %q: %w", kind, c.Name, err)
			}

			entry := lockfile.Entry{
				Name:    c.Name,
				Spec:    c.NewSpec,
				Version: version,
				Source:  lockfile.SourceRegistry,
				Artifacts: []lockfile.Artifact{{
					Platform:    platform,
					Arch:        arch,
					URL:         art.URL,
					Checksum:    art.Checksum,
					Size:        art.Size,
					Strip:       art.Strip,
					LinkBinFrom: art.LinkBinFrom,
				}},
			}

			if c.Section == lockfile.SectionLanguage {
				res.Languages[c.Name] = entry
			} else {
				res.Tools[c.Name] = entry
			}

		case lockfile.SectionPackage:
			// Ecosystem lockfiles aren't registry artifacts; record the path so
			// the lock tracks the toml. Digest drift is out of scope (ION-16).
			res.Packages[c.Name] = lockfile.EcoLockRef{
				Ecosystem: c.Name,
				Path:      c.NewSpec,
			}
		}
	}

	return res, nil
}

// buildMeta assembles the lock's [meta] block from the current config, the
// running binary's version, and the resolved registry index version.
func buildMeta(cfg config.Config, client *registry.Client) lockfile.Meta {
	platform := runtime.GOOS + "-" + runtime.GOARCH // e.g. "darwin-arm64"
	return lockfile.Meta{
		Schema:            lockfile.CurrentSchema,
		Project:           cfg.Compendium.Name,
		CompendiumVersion: buildinfo.Version,
		Platforms:         []string{platform},
		Registry: lockfile.RegistryMeta{
			Source:       lockfile.SourceRegistry,
			IndexVersion: client.IndexVersion(),
		},
	}
}

// unionPlatforms appends any platform in `add` not already in `base`,
// preserving order. Grow-only: platforms are never removed here — that is a
// deliberate future command, not a side effect of install (see ION-16).
func unionPlatforms(base, add []string) []string {
	for _, p := range add {
		if !slices.Contains(base, p) {
			base = append(base, p)
		}
	}
	return base
}

var frozen bool

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVar(&frozen, "frozen", false, "fail if compendium.toml has drifted from compendium.lock")
	installCmd.Flags().BoolVar(&frozen, "locked", false, "alias for --frozen")
}
