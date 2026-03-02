package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	URL      string `yaml:"url"`
	Token    string `yaml:"token"`
	Insecure bool   `yaml:"insecure"`
}

// Path returns the config file path.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "hac", "config.yaml"), nil
}

func Load() (*Config, error) {
	cfg := &Config{}

	path, err := Path()
	if err == nil {
		data, err := os.ReadFile(path)
		if err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parsing %s: %w", path, err)
			}
		}
	}

	if v := os.Getenv("HAC_URL"); v != "" {
		cfg.URL = v
	}
	if v := os.Getenv("HAC_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("HAC_INSECURE"); v == "1" || v == "true" {
		cfg.Insecure = true
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("Home Assistant URL not set (config url or HAC_URL)")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("Home Assistant token not set (config token or HAC_TOKEN)")
	}

	return cfg, nil
}
