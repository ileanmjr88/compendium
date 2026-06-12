// Package lockfile loads and validates compendium.lock files
package lockfile

const (
	CurrentSchema  = 1
	SourceRegistry = "registry"
)

type Lockfile struct {
	Meta      Meta            `toml:"meta"`
	Languages []Entry         `toml:"language"`
	Tools     []Entry         `toml:"tool"`
	Packages  PackagesSection `toml:"packages"`
}

type Meta struct {
	Schema            int          `toml:"schema"`
	Project           string       `toml:"project"`
	CompendiumVersion string       `toml:"compendium_version"`
	Platforms         []string     `toml:"platforms"`
	Registry          RegistryMeta `toml:"registry"`
}

type RegistryMeta struct {
	Source       string `toml:"source"`
	IndexVersion string `toml:"index_version"`
}

type Entry struct {
	Name      string     `toml:"name"`
	Spec      string     `toml:"spec"`
	Version   string     `toml:"version"`
	Source    string     `toml:"source"`
	Artifacts []Artifact `toml:"artifacts"`
}

type Artifact struct {
	Platform    string `toml:"platform"`
	Arch        string `toml:"arch"`
	URL         string `toml:"url"`
	Checksum    string `toml:"checksum"`
	Size        int64  `toml:"size"`
	Strip       int    `toml:"strip"`
	LinkBinFrom string `toml:"link_bin_from,omitempty"`
}

type PackagesSection struct {
	Lockfiles []EcoLockRef `toml:"lockfile"`
}

type EcoLockRef struct {
	Ecosystem string `toml:"ecosystem"`
	Path      string `toml:"path"`
	Digest    string `toml:"digest"`
	Baseline  string `toml:"baseline,omitempty"`
}
