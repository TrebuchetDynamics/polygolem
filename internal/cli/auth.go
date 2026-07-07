package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/clob"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/authclobprobe"
	"github.com/TrebuchetDynamics/polygolem/internal/workflows/authstatus"
	"github.com/spf13/cobra"
)

func newAuthStatusCommand(jsonOut bool) *cobra.Command {
	w := newWire(jsonOut)
	var checkDepositKey bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check authentication readiness and API key status",
		Long: `Inspects the current SIGNER_PRIVATE_KEY and reports:
  - EOA address and deposit wallet address
  - Whether the deposit wallet is deployed
  - Whether EOA-bound CLOB credentials are present
  - Whether the setup is ready for deposit-wallet trading

Use --check-deposit-key to test whether the configured CLOB key works for
trading-readiness checks (makes a live network call). Without this flag, the
check is faster but may report a stale key as existing.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := authstatus.New(authstatus.Config{
				PrivateKey: requirePrivateKey,
				Relayer: func(context.Context) (authstatus.RelayerReader, error) {
					return relayerClientFromEnv()
				},
				CLOB:          clob.NewClient(clobBaseURL, nil),
				L2Credentials: clobL2CredentialsFromEnv,
			}).Status(cmd.Context(), authstatus.Request{CheckDepositKey: checkDepositKey})
			if err != nil {
				return err
			}
			return w.printJSON(cmd, result)
		},
	}

	cmd.Flags().BoolVar(&checkDepositKey, "check-deposit-key", false, "make a live network call to verify configured CLOB credentials")
	return cmd
}

func newAuthCLOBProbeCommand(jsonOut bool) *cobra.Command {
	w := newWire(jsonOut)

	cmd := &cobra.Command{
		Use:   "clob-probe",
		Short: "Probe configured CLOB L2 credentials with read-only calls",
		Long: `Uses configured CLOB L2 HMAC credentials to run authenticated,
read-only CLOB checks without creating or deriving an API key. The probe calls
only GET /data/orders, GET /data/trades, and GET /balance-allowance.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := authclobprobe.New(authclobprobe.Config{
				PrivateKey:    requirePrivateKey,
				L2Credentials: clobL2CredentialsFromEnv,
				CLOB:          w.clob,
			}).Probe(cmd.Context())
			if err != nil {
				return err
			}
			return w.printJSON(cmd, result)
		},
	}

	return cmd
}

func warnIfNoDepositKey(ctx context.Context, stderr io.Writer, privateKey string) {
	signer, err := auth.NewPrivateKeySigner(privateKey, 137)
	if err != nil {
		return
	}
	owner := signer.Address()

	depositWallet, err := auth.MakerAddressForSignatureType(owner, 137, 3)
	if err != nil {
		return
	}

	var deployed bool
	if rc, err := relayerClientFromEnv(); err == nil {
		deployed, _ = rc.IsDeployed(ctx, owner)
	}
	if !deployed {
		return
	}

	if key, ok := clobL2CredentialsFromEnv(); ok && key.Validate() == nil {
		return
	}

	c := clob.NewClient(clobBaseURL, nil)
	_, err = c.DeriveAPIKeyForAddress(ctx, privateKey, depositWallet)
	if err == nil {
		return
	}

	fmt.Fprintf(stderr, "\n⚠️  WARNING: Deposit wallet %s is deployed but no CLOB L2 API key was found.\n", depositWallet)
	fmt.Fprintf(stderr, "   Polymarket login signs with the EOA; the deposit wallet remains the trading wallet.\n")
	fmt.Fprintf(stderr, "   Run:\n")
	fmt.Fprintf(stderr, "   → polygolem auth login\n")
	fmt.Fprintf(stderr, "   → polygolem builder auto   # or: polygolem clob create-api-key\n")
	fmt.Fprintf(stderr, "   → docs/ONBOARDING.md\n\n")
}

func newAuthExportKeyCommand(jsonOut bool) *cobra.Command {
	w := newWire(jsonOut)
	var confirm string
	var confirmAddressSuffix string

	cmd := &cobra.Command{
		Use:   "export-key",
		Short: "HIGH RISK: display private key for wallet import",
		Long: `Displays the current SIGNER_PRIVATE_KEY and derived addresses
in formats suitable for wallet import. This is useful when a bot/agent
generated the key and the user needs to import it into MetaMask/Rabby/etc.
for the one-time Polymarket browser signup.

SECURITY WARNING: The private key will be printed to your terminal.
Anyone with access to your screen or shell history can steal your funds.
Use this only in a secure environment and clear your terminal history after.
This command requires both a typed confirmation token and the last six hex
characters of the EOA address to reduce accidental key disclosure.

Recommended flow for bot-generated keys:
  1. Run this command in a secure terminal
  2. Import the private key into a temporary wallet (MetaMask mobile, fresh browser profile)
  3. Connect to polymarket.com and complete signup
  4. Remove the imported account from the wallet
  5. Clear terminal history: history -c && clear`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			privateKey, err := privateKeyFromEnv()
			if err != nil {
				return err
			}

			signer, err := auth.NewPrivateKeySigner(privateKey, 137)
			if err != nil {
				return fmt.Errorf("init signer: %w", err)
			}
			owner := signer.Address()
			expectedSuffix := addressSuffix(owner, 6)
			gotSuffix := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(confirmAddressSuffix), "0x"))
			if confirm != "EXPORT_PRIVATE_KEY" || gotSuffix != expectedSuffix {
				return fmt.Errorf("this command prints your private key to the terminal; pass --confirm EXPORT_PRIVATE_KEY --confirm-address-suffix %s to proceed", expectedSuffix)
			}

			depositWallet, err := auth.MakerAddressForSignatureType(owner, 137, 3)
			if err != nil {
				return fmt.Errorf("derive deposit wallet: %w", err)
			}

			stderr := cmd.ErrOrStderr()
			fmt.Fprintf(stderr, "\n⚠️  SECURITY WARNING: Private key exposed below. Clear your terminal history after.\n\n")

			return w.printJSON(cmd, map[string]string{
				"eoaAddress":    owner,
				"depositWallet": depositWallet,
				"privateKey":    privateKey,
				"warning":       "Clear terminal history after import: history -c && clear",
			})
		},
	}

	cmd.Flags().StringVar(&confirm, "confirm", "", "must be exactly EXPORT_PRIVATE_KEY to print the private key")
	cmd.Flags().StringVar(&confirmAddressSuffix, "confirm-address-suffix", "", "last 6 hex characters of the EOA address")
	return cmd
}

func addressSuffix(address string, n int) string {
	clean := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(address), "0x"))
	if len(clean) <= n {
		return clean
	}
	return clean[len(clean)-n:]
}

func warnIfNoDepositKeySimple(stderr io.Writer, privateKey string) {
	signer, err := auth.NewPrivateKeySigner(privateKey, 137)
	if err != nil {
		return
	}
	owner := signer.Address()

	depositWallet, err := auth.MakerAddressForSignatureType(owner, 137, 3)
	if err != nil {
		return
	}

	fmt.Fprintf(stderr, "\nℹ️  Note: If this is your first time using Polymarket with this key,\n")
	fmt.Fprintf(stderr, "   run `polygolem auth login` so the EOA signs the Polymarket SIWE\n")
	fmt.Fprintf(stderr, "   message and the deposit wallet is registered as the trading wallet.\n")
	fmt.Fprintf(stderr, "   Deposit wallet: %s\n", depositWallet)
	fmt.Fprintf(stderr, "   See: docs/ONBOARDING.md\n\n")
}
