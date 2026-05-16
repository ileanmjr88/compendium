package installer

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ileanmjr88/compendium/internal/config"
	"github.com/ileanmjr88/compendium/internal/env"
	"github.com/ileanmjr88/compendium/internal/extract"
	"github.com/ileanmjr88/compendium/internal/registry"
	"github.com/ileanmjr88/compendium/internal/ui"
)

type InstallItem struct {
	Name     string
	Version  string
	Platform string
	Arch     string
	Kind     string
}

func Resolve(cfg config.Config) []InstallItem {
	items := []InstallItem{}
	platform := runtime.GOOS
	arch := runtime.GOARCH

	// Warn if both c and cpp are set - one compiler handles both drivers
	skipCpp := false
	if cfg.Languages["c"] != "" && cfg.Languages["cpp"] != "" {
		ui.Print(ui.Warning, "both c and cpp set in [languages]", "using c; a single compiler ships both drivers")
		skipCpp = true
	}

	// Languages
	for name, version := range cfg.Languages {
		if name == "cpp" && skipCpp {
			continue
		}
		item := InstallItem{Arch: arch, Kind: "languages", Platform: platform}

		if strings.Contains(version, "@") {
			parts := strings.SplitN(version, "@", 2)
			item.Name = parts[0]
			item.Version = parts[1]
		} else {
			item.Name = name
			item.Version = version
		}

		items = append(items, item)
	}

	// Tools
	for name, version := range cfg.Tools {
		items = append(items, InstallItem{Arch: arch, Kind: "tools", Platform: platform, Name: name, Version: version})
	}

	return items
}

func FilterInstalled(items []InstallItem, paths *env.Paths) []InstallItem {
	newItems := []InstallItem{}

	// Group items by kind for display
	ui.Print(ui.Action, "resolving languages", "")
	for _, item := range items {
		if item.Kind != "languages" {
			continue
		}
		if paths.CheckDir(item.Kind, item.Name, item.Version) {
			ui.PrintDetail(ui.Success, item.Name+" "+item.Version, "already installed")
		} else {
			ui.PrintDetail(ui.Action, item.Name+" "+item.Version, "not installed")
			newItems = append(newItems, item)
		}
	}

	ui.Print(ui.Action, "resolving tools", "")
	for _, item := range items {
		if item.Kind != "tools" {
			continue
		}
		if paths.CheckDir(item.Kind, item.Name, item.Version) {
			ui.PrintDetail(ui.Success, item.Name+" "+item.Version, "already installed")
		} else {
			ui.PrintDetail(ui.Action, item.Name+" "+item.Version, "not installed")
			newItems = append(newItems, item)
		}
	}
	fmt.Println()

	return newItems
}

func DownloadVerify(items []InstallItem, paths *env.Paths, client *registry.Client) error {
	tmpDir := filepath.Join(paths.Root, "tmp")
	_ = os.MkdirAll(tmpDir, 0755)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	for _, item := range items {
		if err := processItem(item, client, tmpDir, paths); err != nil {
			return err
		}
	}

	return nil
}

func processItem(item InstallItem, client *registry.Client, tmpDir string, paths *env.Paths) error {
	_ = os.MkdirAll(tmpDir, 0755)

	// Lookup the artifact (gets URL, checksum, size)
	artifact, _, err := client.Lookup(item.Kind, item.Name, item.Version, item.Platform, item.Arch)
	if err != nil {
		return fmt.Errorf("client lookup: %w", err)
	}

	// Download the tarball from artifact.URL
	resp, err := http.Get(artifact.URL)
	if err != nil {
		return fmt.Errorf("fetching artifact: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	ext := archiveExt(artifact.URL)
	filePath := filepath.Join(tmpDir, item.Name+"-"+item.Version+ext)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer func() { _ = file.Close() }()

	bar := ui.ProgressBar(resp.ContentLength, item.Name)
	if _, err := io.Copy(io.MultiWriter(file, bar), resp.Body); err != nil {
		return fmt.Errorf("downloading %s: %w", item.Name, err)
	}

	// Verify checksum matches artifact.Checksum
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file for checksum: %w", err)
	}
	defer func() { _ = f.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("hashing %s: %w", item.Name, err)
	}

	checksum := fmt.Sprintf("sha256:%x", hasher.Sum(nil))
	if checksum != artifact.Checksum {
		ui.Print(ui.Fail, "checksum mismatch", item.Name)
		return fmt.Errorf("checksum mismatch for %s", item.Name)
	}
	ui.Print(ui.Success, "checksum ok", checksum[:23]+"...")

	// Extract to the right path using paths + item.Kind
	var destDir string
	if item.Kind == "languages" {
		destDir = paths.LanguageDir(item.Name, item.Version)
	} else {
		destDir = paths.ToolDir(item.Name, item.Version)
	}

	if isArchive(filePath) {
		ui.Print(ui.Action, "extracting "+item.Name, "")
		err = extract.Extract(filePath, destDir, artifact.Strip)
		if err != nil {
			ui.Print(ui.Fail, "extracting "+item.Name, err.Error())
			return fmt.Errorf("extracting %s: %w", item.Name, err)
		}
	} else {
		ui.Print(ui.Action, "installing "+item.Name, "")
		err = installBinary(filePath, destDir, item.Name)
		if err != nil {
			ui.Print(ui.Fail, "installing "+item.Name, err.Error())
			return fmt.Errorf("installing %s: %w", item.Name, err)
		}
	}
	// Normalize: ensure tools have a bin/ directory
	if item.Kind == "tools" {
		binDir := filepath.Join(destDir, "bin")
		if _, err := os.Stat(binDir); os.IsNotExist(err) {
			_ = os.MkdirAll(binDir, 0755)
			entries, _ := os.ReadDir(destDir)
			for _, e := range entries {
				if e.Name() == "bin" {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
					_ = os.Rename(filepath.Join(destDir, e.Name()), filepath.Join(binDir, e.Name()))
				}
			}
		}
	}

	ui.Print(ui.Success, "installed "+item.Name, item.Version)

	return nil
}

func isArchive(path string) bool {
	for _, ext := range []string{".tar.gz", ".tgz", ".tar.xz", ".tar.zst", ".tar.bz2", ".zip"} {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func archiveExt(url string) string {
	for _, ext := range []string{".tar.gz", ".tar.xz", ".tar.zst", ".tar.bz2", ".tgz", ".zip"} {
		if strings.HasSuffix(url, ext) {
			return ext
		}
	}
	return filepath.Ext(url)
}

func installBinary(src, destDir, name string) error {
	if err := os.MkdirAll(filepath.Join(destDir, "bin"), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	dest := filepath.Join(destDir, "bin", name)
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source: %w", err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating destination: %w", err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copying binary: %w", err)
	}

	return os.Chmod(dest, 0755)
}

func satisfies(pinned, required string) bool {
	return pinned == required || strings.HasPrefix(pinned, required+".")
}

func ValidateDeps(items []InstallItem, client *registry.Client, cfg config.Config) error {
	pinned := make(map[string]string, len(items))
	for _, it := range items {
		pinned[it.Name] = it.Version
	}

	for _, it := range items {
		_, depends, err := client.Lookup(it.Kind, it.Name, it.Version, it.Platform, it.Arch)
		if err != nil {
			return fmt.Errorf("validating %s %s: %w", it.Name, it.Version, err)
		}

		for depName, depVersion := range depends {
			have, ok := pinned[depName]
			if !ok {
				return fmt.Errorf(
					"%s %s requires %s %s; add %s = %q to compendium.toml",
					it.Name, it.Version, depName, depVersion, depName, depVersion,
				)
			}

			if !satisfies(have, depVersion) {
				return fmt.Errorf(
					"%s %s requires %s %s, but compendium.toml pins %s = %q",
					it.Name, it.Version, depName, depVersion, depName, have,
				)
			}
		}
	}
	return nil
}
