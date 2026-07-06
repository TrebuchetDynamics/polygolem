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
		Use:   "polygolem",
		Short: "Go CLI and SDK for safe Polymarket V2 deposit-wallet trading",
		Long: `polygolem is a Go CLI and SDK for safe Polymarket V2 deposit-wallet trading.

Read-only by default: market data, discovery, streaming, order books, data
analytics, health checks, and diagnostics need no credentials. Authenticated
paths are opt-in only when SIGNER_PRIVATE_KEY is set, and every command that
moves funds gates on an explicit cap and a typed --confirm token.

Start here (no credentials needed):
  polygolem health                 # is the API reachable?
  polygolem discover search --query "Will BTC" --limit 5
  polygolem orderbook get --token-id <id>
  polygolem paper reset --cash 100 # simulate trading with zero risk

When you are ready to trade with real funds, read the safety model first:
  docs/SAFETY.md and docs/SAFE-HAPPY-PATH.md

Every command accepts --json for a stable {ok,version,data,meta} envelope.`,
		Example: `  # Read-only: check reachability and read a live order book
  polygolem health --json
  polygolem orderbook get --token-id 7132104567... --json

  # Paper trade with no wallet and no risk
  polygolem paper reset --cash 100
  polygolem paper trade --asset BTC --interval 5m --side up --size 1

  # Live: the smallest capped order (needs SIGNER_PRIVATE_KEY + a funded deposit wallet)
  POLYGOLEM_MAX_LIVE_ORDER_USD=1 polygolem clob create-order \
    --token <id> --side buy --price 0.40 --size 2`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(opts.Stdout)
	root.SetErr(opts.Stderr)
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "emit JSON output")

	// Command groups make the safety posture visible in `polygolem --help`:
	// read-only commands are separated from the ones that move real funds.
	const (
		groupStart   = "getting-started"
		groupReadNly = "read-only"
		groupPaper   = "paper"
		groupLive    = "live"
	)
	root.AddGroup(
		&cobra.Group{ID: groupStart, Title: "Getting started & diagnostics:"},
		&cobra.Group{ID: groupReadNly, Title: "Market data & research (read-only, no credentials):"},
		&cobra.Group{ID: groupPaper, Title: "Paper trading (no risk):"},
		&cobra.Group{ID: groupLive, Title: "Trading & wallet (live — credentials required):"},
	)
	addTo := func(groupID string, cmds ...*cobra.Command) {
		for _, c := range cmds {
			c.GroupID = groupID
			root.AddCommand(c)
		}
	}

	addTo(groupStart, &cobra.Command{
		Use: "version", Short: "Print version", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput {
				return writeCommandJSON(cmd, map[string]string{"version": opts.Version})
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "polygolem %s\n", opts.Version)
			return err
		},
	})

	addTo(groupStart, &cobra.Command{
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

	addTo(groupStart, healthCmd(jsonOutput), diagCmd(jsonOutput, opts.Version), driftCmd(jsonOutput), liveCmd(opts.Version))
	addTo(groupReadNly,
		discoverCmd(jsonOutput),
		eventsCmd(jsonOutput),
		orderbookCmd(jsonOutput),
		marketDataCmd(jsonOutput),
		streamCmd(jsonOutput),
		dataCmd(jsonOutput),
		intelCmd(jsonOutput),
	)
	addTo(groupPaper, paperCmd(jsonOutput))

	authCmd := commandGroup("auth", "Inspect authentication readiness",
		newAuthStatusCommand(jsonOutput),
	)
	authCmd.Long = `Authentication readiness and login for Polymarket.

Read-only: 'auth status' and 'auth clob-probe' report credential readiness.
Live: 'auth login' and 'auth headless-onboard' sign SIWE and mint/persist V2
relayer credentials. 'auth export-key' prints your private key and is
double-confirmed — avoid it unless importing into a temporary browser wallet.`
	authCmd.AddCommand(newAuthCLOBProbeCommand(jsonOutput))
	authCmd.AddCommand(newAuthLoginCommand(jsonOutput))
	authCmd.AddCommand(newAuthHeadlessOnboardCommand(jsonOutput))
	authCmd.AddCommand(newAuthExportKeyCommand(jsonOutput))
	addTo(groupLive,
		authCmd,
		depositWalletCmd(jsonOutput),
		clobCmd(jsonOutput),
		bridgeCmd(jsonOutput),
		relayerCmd(jsonOutput),
		newBuilderCommand(jsonOutput),
	)
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
