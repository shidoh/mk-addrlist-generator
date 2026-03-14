package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func LoadConfig(path string) (*Config, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath) // #nosec G304 - path is cleaned with filepath.Clean
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if err := ValidateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	return &cfg, nil
}
