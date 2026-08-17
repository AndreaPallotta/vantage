package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AndreaPallotta/vantage/internal/models"
)

// Config represents Vantage configuration settings.
type Config struct {
	ActiveSpace     string               `json:"active_space"`
	Port            int                  `json:"port"`
	AutoOpen        bool                 `json:"auto_open"`
	RefreshSec      int                  `json:"refresh_sec"`
	IncludeForks    bool                 `json:"include_forks"`
	IncludeArchived bool                 `json:"include_archived"`
	Spaces          []models.SpaceConfig `json:"spaces"`
}

// DefaultConfig returns defaults with the primary GitHub space configured.
func DefaultConfig() *Config {
	return &Config{
		ActiveSpace:     "all",
		Port:            8080,
		AutoOpen:        true,
		RefreshSec:      30,
		IncludeForks:    false,
		IncludeArchived:  false,
		Spaces: []models.SpaceConfig{
			{
				ID:        "github-andrea",
				Name:      "GitHub (AndreaPallotta)",
				Platform:  models.PlatformGitHub,
				BaseURL:   "https://api.github.com",
				Namespace: "AndreaPallotta",
				TokenEnv:  "GITHUB_TOKEN",
			},
		},
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

	if len(cfg.Spaces) == 0 {
		cfg.Spaces = DefaultConfig().Spaces
	}

	// Environment variable overrides
	if envSpace := os.Getenv("VANTAGE_SPACE"); envSpace != "" {
		cfg.ActiveSpace = envSpace
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

// ResolveToken determines the token for a specific space config.
func ResolveToken(sc models.SpaceConfig) string {
	if sc.Token != "" {
		return sc.Token
	}

	if sc.TokenEnv != "" {
		if val := os.Getenv(sc.TokenEnv); val != "" {
			return val
		}
	}

	if sc.Platform == models.PlatformGitHub {
		if val := os.Getenv("GITHUB_TOKEN"); val != "" {
			return val
		}
		if val := os.Getenv("GH_TOKEN"); val != "" {
			return val
		}
		if tok, err := getGhCliToken(); err == nil && tok != "" {
			return tok
		}
	} else if sc.Platform == models.PlatformGitLab {
		if val := os.Getenv("GITLAB_TOKEN"); val != "" {
			return val
		}
		if val := os.Getenv("GL_TOKEN"); val != "" {
			return val
		}
	}

	return ""
}

func getGhCliToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
