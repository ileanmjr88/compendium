package lockfile

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

func Load(path string) (*Lockfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var lf Lockfile
	if err := toml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if lf.Meta.Schema > CurrentSchema {
		return nil, &SchemaVersionError{Path: path, Found: lf.Meta.Schema, Max: CurrentSchema}
	}

	if err := lf.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return &lf, nil
}
