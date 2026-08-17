package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config represents Vantage configuration settings.
type Config struct {
	Token          string `json:"token,omitempty"`
	Space          string `json:"space,omitempty"`
	Port           int    `json:"port"`
	AutoOpen       bool   `json:"auto_open"`
	RefreshSec     int    `json:"refresh_sec"`
	IncludeForks   bool   `json:"include_forks"`
	IncludeArchived bool  `json:"include_archived"`
}

// DefaultConfig returns reasonable defaults for Vantage.
func DefaultConfig() *Config {
	return &Config{
		Port:           8080,
		AutoOpen:       true,
		RefreshSec:     30,
		IncludeForks:   false,
		IncludeArchived: false,
	}
}

// ConfigPath returns the path to ~/.vantage/config.json
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".vantage", "config.json"), nil
}

// Load reads config from ~/.vantage/config.json if it exists, falling back to defaults.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	path, err := ConfigPath()
	if err == nil {
		if data, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(data, cfg)
		}
	}

	// Environment variable overrides
	if envToken := os.Getenv("GITHUB_TOKEN"); envToken != "" {
		cfg.Token = envToken
	} else if envToken := os.Getenv("GH_TOKEN"); envToken != "" {
		cfg.Token = envToken
	}

	if envSpace := os.Getenv("VANTAGE_SPACE"); envSpace != "" {
		cfg.Space = envSpace
	}

	// Fallback to `gh auth token` if no token found
	if cfg.Token == "" {
		if token, err := getGhCliToken(); err == nil && token != "" {
			cfg.Token = token
		}
	}

	return cfg, nil
}

// Save writes config to ~/.vantage/config.json.
func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func getGhCliToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
