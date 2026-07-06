package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/TrebuchetDynamics/polygolem/internal/clob"
	"github.com/TrebuchetDynamics/polygolem/internal/dataapi"
	"github.com/TrebuchetDynamics/polygolem/internal/gamma"
	"github.com/TrebuchetDynamics/polygolem/internal/marketdiscovery"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/localpreflight"
	"github.com/spf13/cobra"
)

const (
	gammaBaseURL        = "https://gamma-api.polymarket.com"
	clobBaseURL         = "https://clob.polymarket.com"
	dataBaseURL         = "https://data-api.polymarket.com"
	marketStreamBaseURL = "wss://ws-subscriptions-clob.polymarket.com/ws/market"
	userStreamBaseURL   = "wss://ws-subscriptions-clob.polymarket.com/ws/user"
)

type Options struct {
	Version string
	Stdout  io.Writer
	Stderr  io.Writer
}

type wire struct {
	gamma    *gamma.Client
	clob     *clob.Client
	data     *dataapi.Client
	discover *marketdiscovery.Service
	jsonOut  bool
}

func (w *wire) printJSON(cmd *cobra.Command, v interface{}) error {
	return writeCommandJSON(cmd, v)
}

func newWire(jsonOut bool) *wire {
	clobURL := firstNonEmptyCLI(firstEnv("POLYMARKET_CLOB_URL", "CLOB_URL"), clobBaseURL)
	clobClient := clob.NewClient(clobURL, nil)
	if key, ok := clobL2CredentialsFromEnv(); ok {
		clobClient.SetL2Credentials(key)
	}

	return &wire{
		gamma:    gamma.NewClient(gammaBaseURL, nil),
		clob:     clobClient,
		data:     dataapi.NewClient(dataBaseURL, nil),
		discover: marketdiscovery.New(gamma.NewClient(gammaBaseURL, nil), clob.NewClient(clobURL, nil)),
		jsonOut:  jsonOut,
	}
}

func NewRootCommand(opts Options) *cobra.Command {
	if opts.Version == "" {
		opts.Version = "dev"
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	var jsonOutput bool

	root := &cobra.Command{
		Use:           "polygolem",
		Short:         "Safe Polymarket SDK and CLI for Go",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(opts.Stdout)
	root.SetErr(opts.Stderr)
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "emit JSON output")

	root.AddCommand(&cobra.Command{
		Use: "version", Short: "Print version", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput {
				return writeCommandJSON(cmd, map[string]string{"version": opts.Version})
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "polygolem %s\n", opts.Version)
			return err
		},
	})

	root.AddCommand(&cobra.Command{
		Use: "preflight", Short: "Inspect local CLI readiness", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := localpreflight.New(localpreflight.Config{Version: opts.Version, BuilderCode: builderCodeFromFlagOrEnv("")})
			result := runner.Run(cmd.Context())
			if jsonOutput {
				return writeCommandJSON(cmd, result)
			}
			return localpreflight.WriteText(cmd.OutOrStdout(), result)
		},
	})

	root.AddCommand(discoverCmd(jsonOutput))
	root.AddCommand(orderbookCmd(jsonOutput))
	root.AddCommand(clobCmd(jsonOutput))
	root.AddCommand(dataCmd(jsonOutput))
	root.AddCommand(intelCmd(jsonOutput))
	root.AddCommand(diagCmd(jsonOutput, opts.Version))
	root.AddCommand(driftCmd(jsonOutput))
	root.AddCommand(healthCmd(jsonOutput))
	root.AddCommand(eventsCmd(jsonOutput))
	root.AddCommand(bridgeCmd(jsonOutput))
	root.AddCommand(marketDataCmd(jsonOutput))
	root.AddCommand(streamCmd(jsonOutput))
	root.AddCommand(depositWalletCmd(jsonOutput))
	root.AddCommand(relayerCmd(jsonOutput))
	root.AddCommand(newBuilderCommand(jsonOutput))
	root.AddCommand(paperCmd(jsonOutput))
	authCmd := commandGroup("auth", "Inspect authentication readiness",
		newAuthStatusCommand(jsonOutput),
	)
	authCmd.AddCommand(newAuthCLOBProbeCommand(jsonOutput))
	authCmd.AddCommand(newAuthLoginCommand(jsonOutput))
	authCmd.AddCommand(newAuthHeadlessOnboardCommand(jsonOutput))
	authCmd.AddCommand(newAuthExportKeyCommand(jsonOutput))
	root.AddCommand(authCmd)
	root.AddCommand(liveCmd(opts.Version))
	installJSONContract(root)
	return root
}
func commandGroup(use, short string, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{Use: use, Short: short, Args: cobra.NoArgs,
		Annotations: map[string]string{commandGroupAnnotation: "true"},
		RunE:        func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}
	cmd.AddCommand(children...)
	return cmd
}

func skeleton(use string) *cobra.Command {
	return &cobra.Command{Use: use, Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonEnabled(cmd) {
				return fmt.Errorf("%s: not implemented", commandName(cmd))
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s: not implemented\n", cmd.CommandPath())
			return err
		},
	}
}
