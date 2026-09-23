package config

import (
	"os"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	_ = os.Unsetenv("APP_ENV")
	_ = os.Unsetenv("HTTP_PORT")

	cfg := Load()
	if cfg.AppEnv == "" {
		t.Errorf("expected default AppEnv, got empty string")
	}
	if cfg.HTTPPort != "8080" {
		t.Errorf("expected default HTTPPort 8080, got %s", cfg.HTTPPort)
	}
}

func TestLoadConfigCustom(t *testing.T) {
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("HTTP_PORT", "9999")
	defer func() {
		_ = os.Unsetenv("APP_ENV")
		_ = os.Unsetenv("HTTP_PORT")
	}()

	cfg := Load()
	if cfg.AppEnv != "test" {
		t.Errorf("expected AppEnv 'test', got %s", cfg.AppEnv)
	}
	if cfg.HTTPPort != "9999" {
		t.Errorf("expected HTTPPort '9999', got %s", cfg.HTTPPort)
	}
}
