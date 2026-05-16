package config

import (
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"os"
)

func Load(path string) (*Config, *ValidateResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("parsing config: %w", err)
	}

	result := cfg.Validate()
	return &cfg, result, nil
}
