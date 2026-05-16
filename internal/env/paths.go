package env

import (
	"fmt"
	"os"
	"path/filepath"
)

type Paths struct {
	Root      string // $HOME/.local/compendium
	Languages string // Root/languages
	Tools     string // Root/tools
	Envs      string // Root/envs
}

func NewPaths() (*Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving home directory: %w", err)
	}
	root := filepath.Join(home, ".local", "compendium")

	return NewPathsWithRoot(root), nil
}

func NewPathsWithRoot(root string) *Paths {
	return &Paths{
		Root:      root,
		Languages: filepath.Join(root, "languages"),
		Tools:     filepath.Join(root, "tools"),
		Envs:      filepath.Join(root, "envs"),
	}
}

func (p *Paths) LanguageDir(name, version string) string {
	return filepath.Join(p.Languages, name, version)
}

func (p *Paths) ToolDir(name, version string) string {
	return filepath.Join(p.Tools, name, version)
}

func (p *Paths) EnvDir(project, version string) string {
	return filepath.Join(p.Envs, project, version)
}

func (p *Paths) ProjectDir(name string) string {
	return filepath.Join(p.Envs, name)
}

func (p *Paths) EnsureDirs() error {
	for _, dir := range []string{p.Root, p.Languages, p.Tools, p.Envs} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}
	return nil
}

func (p *Paths) CheckDir(kind string, name string, version string) bool {
	path := filepath.Join(p.Root, kind, name, version)

	_, err := os.Stat(path)
	return err == nil
}
