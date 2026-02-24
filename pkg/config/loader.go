package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// validateConfig performs basic validation of the configuration
func validateConfig(config *Config) error {
	if config.Lists == nil {
		return fmt.Errorf("no lists defined in configuration")
	}

	for name, list := range config.Lists {
		if err := validateList(name, &list); err != nil {
			return err
		}
	}

	return nil
}

// validateList validates a single list configuration
func validateList(name string, list *List) error {
	if len(list.URLs) == 0 && len(list.Files) == 0 && len(list.Addresses) == 0 {
		return fmt.Errorf("list %q has no sources configured (urls, files, or addresses)", name)
	}

	if list.Timeout != "" {
		if _, err := time.ParseDuration(list.Timeout); err != nil {
			return fmt.Errorf("invalid timeout in list %q: %w", name, err)
		}
	}

	return nil
}