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
}

func TestLoadWithEnvOverrides(t *testing.T) {
	os.Setenv("GITHUB_TOKEN", "test-token-12345")
	os.Setenv("VANTAGE_SPACE", "TestUser")
	defer func() {
		os.Unsetenv("GITHUB_TOKEN")
		os.Unsetenv("VANTAGE_SPACE")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Token != "test-token-12345" {
		t.Errorf("expected token to be test-token-12345, got %s", cfg.Token)
	}
	if cfg.Space != "TestUser" {
		t.Errorf("expected space to be TestUser, got %s", cfg.Space)
	}
}
