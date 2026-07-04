package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/config"
	"github.com/TrebuchetDynamics/polygolem/internal/modes"
	"github.com/TrebuchetDynamics/polygolem/internal/preflight"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/localpreflight"
	"github.com/spf13/cobra"
)

type liveStatusData struct {
	Allowed       bool             `json:"allowed"`
	Mode          string           `json:"mode"`
	EnvEnabled    bool             `json:"env_enabled"`
	ConfigEnabled bool             `json:"config_enabled"`
	ConfirmLive   bool             `json:"confirm_live"`
	PreflightOK   bool             `json:"preflight_ok"`
	Failures      []modes.Failure  `json:"failures,omitempty"`
	Preflight     preflight.Result `json:"preflight"`
}

func liveCmd(version string) *cobra.Command {
	return commandGroup("live", "Inspect live gate status", newLiveStatusCommand(version))
}

func newLiveStatusCommand(version string) *cobra.Command {
	var confirmLive bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Inspect live gate status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.Options{})
			if err != nil {
				return err
			}
			preflightResult := localpreflight.New(localpreflight.Config{Version: version, BuilderCode: builderCodeFromFlagOrEnv("")}).Run(cmd.Context())
			result := modes.ValidateLiveGates(modes.LiveGateInput{
				EnvEnabled:    liveProfileEnabled(),
				ConfigEnabled: cfg.LiveTradingEnabled,
				ConfirmLive:   confirmLive,
				PreflightOK:   preflightResult.OK,
			})
			data := liveStatusData{
				Allowed:       result.Allowed,
				Mode:          cfg.Mode,
				EnvEnabled:    liveProfileEnabled(),
				ConfigEnabled: cfg.LiveTradingEnabled,
				ConfirmLive:   confirmLive,
				PreflightOK:   preflightResult.OK,
				Failures:      result.Failures,
				Preflight:     preflightResult,
			}
			if jsonEnabled(cmd) {
				return writeCommandJSON(cmd, data)
			}
			return writeLiveStatusText(cmd.OutOrStdout(), data)
		},
	}
	cmd.Flags().BoolVar(&confirmLive, "confirm-live", false, "include the live confirmation gate in status evaluation")
	return cmd
}

func liveProfileEnabled() bool {
	value := strings.TrimSpace(os.Getenv("POLYMARKET_LIVE_PROFILE"))
	return strings.EqualFold(value, "on") || value == "1" || strings.EqualFold(value, "true")
}

func writeLiveStatusText(w io.Writer, data liveStatusData) error {
	status := "blocked"
	if data.Allowed {
		status = "allowed"
	}
	if _, err := fmt.Fprintf(w, "live: %s\n", status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "- mode: %s\n- env_enabled: %t\n- config_enabled: %t\n- confirm_live: %t\n- preflight_ok: %t\n", data.Mode, data.EnvEnabled, data.ConfigEnabled, data.ConfirmLive, data.PreflightOK); err != nil {
		return err
	}
	for _, failure := range data.Failures {
		if _, err := fmt.Fprintf(w, "- failure: %s (%s)\n", failure.Code, failure.Message); err != nil {
			return err
		}
	}
	return nil
}
