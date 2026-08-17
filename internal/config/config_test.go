package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
	if !cfg.AutoOpen {
		t.Errorf("expected default AutoOpen to be true")
	}
	if len(cfg.Spaces) == 0 {
		t.Errorf("expected at least 1 default space")
	}
}

func TestLoadWithEnvOverrides(t *testing.T) {
	os.Setenv("VANTAGE_SPACE", "github-andrea")
	defer func() {
		os.Unsetenv("VANTAGE_SPACE")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ActiveSpace != "github-andrea" {
		t.Errorf("expected active space to be github-andrea, got %s", cfg.ActiveSpace)
	}
}
