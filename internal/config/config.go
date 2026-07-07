package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/modes"
)

type Options struct {
	ConfigPath string
	EnvPrefix  string
}

type Config struct {
	Mode               string
	GammaBaseURL       string
	CLOBBaseURL        string
	RequestTimeout     time.Duration
	LiveTradingEnabled bool
	PaperStatePath     string
}

func Load(opts Options) (Config, error) {
	values := map[string]string{
		"mode":                 "read-only",
		"gamma_base_url":       "https://gamma-api.polymarket.com",
		"clob_base_url":        "https://clob.polymarket.com",
		"request_timeout":      "10s",
		"live_trading_enabled": "false",
		"paper_state_path":     "",
	}
	if opts.ConfigPath != "" {
		fileValues, err := readConfigFile(opts.ConfigPath)
		if err != nil {
			return Config{}, err
		}
		for key, value := range fileValues {
			values[key] = value
		}
	}
	if opts.EnvPrefix == "" {
		opts.EnvPrefix = "POLYMARKET"
	}
	applyEnv(values, opts.EnvPrefix)

	timeout, err := time.ParseDuration(values["request_timeout"])
	if err != nil {
		return Config{}, err
	}
	if timeout <= 0 {
		return Config{}, errors.New("request_timeout must be positive")
	}
	liveTradingEnabled, err := strconv.ParseBool(values["live_trading_enabled"])
	if err != nil {
		return Config{}, fmt.Errorf("live_trading_enabled: %w", err)
	}
	cfg := Config{
		Mode:               values["mode"],
		GammaBaseURL:       values["gamma_base_url"],
		CLOBBaseURL:        values["clob_base_url"],
		RequestTimeout:     timeout,
		LiveTradingEnabled: liveTradingEnabled,
		PaperStatePath:     values["paper_state_path"],
	}
	if cfg.Mode == "" {
		return Config{}, errors.New("mode is required")
	}
	// Reject unknown mode strings (e.g. typos) rather than letting them flow to
	// callers that gate behavior on exact-string comparisons.
	if _, err := modes.Parse(cfg.Mode); err != nil {
		return Config{}, fmt.Errorf("invalid mode: %w", err)
	}
	return cfg, nil
}

func applyEnv(values map[string]string, prefix string) {
	for key := range values {
		if value, ok := os.LookupEnv(prefix + "_" + strings.ToUpper(key)); ok {
			values[key] = value
		}
	}
}

func readConfigFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("%s:%d: expected key: value", path, lineNumber)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("%s:%d: missing key", path, lineNumber)
		}
		values[key] = cleanYAMLScalar(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func cleanYAMLScalar(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return value
}
