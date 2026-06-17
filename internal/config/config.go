package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"

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
	v := viper.New()
	v.SetConfigType("yaml")
	if opts.EnvPrefix == "" {
		opts.EnvPrefix = "POLYMARKET"
	}
	v.SetEnvPrefix(opts.EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetDefault("mode", "read-only")
	v.SetDefault("gamma_base_url", "https://gamma-api.polymarket.com")
	v.SetDefault("clob_base_url", "https://clob.polymarket.com")
	v.SetDefault("request_timeout", "10s")
	v.SetDefault("live_trading_enabled", false)
	v.SetDefault("paper_state_path", "")
	if opts.ConfigPath != "" {
		v.SetConfigFile(opts.ConfigPath)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, err
		}
	}
	timeout, err := time.ParseDuration(v.GetString("request_timeout"))
	if err != nil {
		return Config{}, err
	}
	if timeout <= 0 {
		return Config{}, errors.New("request_timeout must be positive")
	}
	cfg := Config{
		Mode:               v.GetString("mode"),
		GammaBaseURL:       v.GetString("gamma_base_url"),
		CLOBBaseURL:        v.GetString("clob_base_url"),
		RequestTimeout:     timeout,
		LiveTradingEnabled: v.GetBool("live_trading_enabled"),
		PaperStatePath:     v.GetString("paper_state_path"),
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
