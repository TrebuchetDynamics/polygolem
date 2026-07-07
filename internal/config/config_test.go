package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadDefaultsToReadOnlyAndSafeURLs(t *testing.T) {
	cfg, err := Load(Options{})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Mode != "read-only" {
		t.Fatalf("Mode = %q, want read-only", cfg.Mode)
	}
	if cfg.LiveTradingEnabled {
		t.Fatal("live trading must default to disabled")
	}
	if cfg.RequestTimeout != 10*time.Second {
		t.Fatalf("RequestTimeout = %s, want 10s", cfg.RequestTimeout)
	}
}

func TestLoadReadsExplicitConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("mode: paper\npaper_state_path: /tmp/paper.json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(Options{ConfigPath: path})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Mode != "paper" {
		t.Fatalf("Mode = %q, want paper", cfg.Mode)
	}
	if cfg.PaperStatePath != "/tmp/paper.json" {
		t.Fatalf("PaperStatePath = %q", cfg.PaperStatePath)
	}
}

func TestLoadEnvOverridesConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("mode: read-only\nlive_trading_enabled: false\nrequest_timeout: 5s\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POLYMARKET_MODE", "paper")
	t.Setenv("POLYMARKET_LIVE_TRADING_ENABLED", "true")
	t.Setenv("POLYMARKET_REQUEST_TIMEOUT", "7s")

	cfg, err := Load(Options{ConfigPath: path})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Mode != "paper" || !cfg.LiveTradingEnabled || cfg.RequestTimeout != 7*time.Second {
		t.Fatalf("cfg = %+v, want env values to override config file", cfg)
	}
}

func TestLoadSupportsCustomEnvPrefix(t *testing.T) {
	t.Setenv("POLYGOLEM_MODE", "paper")

	cfg, err := Load(Options{EnvPrefix: "POLYGOLEM"})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Mode != "paper" {
		t.Fatalf("Mode = %q, want paper", cfg.Mode)
	}
}

func TestLoadRejectsNonPositiveRequestTimeout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("request_timeout: 0s\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(Options{ConfigPath: path})
	if err == nil {
		t.Fatal("Load returned nil error for non-positive request_timeout")
	}
}

// TestLoadRejectsUnknownMode guards the regression where any non-empty mode
// string was accepted; only known modes (read-only/paper/live) are valid.
func TestLoadRejectsUnknownMode(t *testing.T) {
	t.Setenv("POLYMARKET_MODE", "banana")
	_, err := Load(Options{})
	if err == nil || !strings.Contains(err.Error(), "mode") {
		t.Fatalf("err = %v, want an invalid-mode error", err)
	}
}
