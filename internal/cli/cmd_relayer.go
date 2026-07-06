package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

func relayerCmd(jsonOut bool) *cobra.Command {
	cmd := commandGroup("relayer", "Inspect Polymarket relayer state")
	cmd.Long = `Inspect Polymarket V2 relayer state. Read-only.

Look up a relayer transaction by id to see its state and on-chain hash. The
wallet lifecycle mutations the relayer sponsors (deploy, batch, approvals) are
driven from 'polygolem deposit-wallet', not here.`
	cmd.Example = `  polygolem relayer transaction <tx-id> --json`
	cmd.AddCommand(relayerTransactionCmd(jsonOut))
	return cmd
}

func relayerTransactionCmd(jsonOut bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transaction <tx-id>",
		Short: "Get relayer transaction state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			txID := strings.TrimSpace(args[0])
			tx, err := depositWalletReadsRunner(cmd, nil).Transaction(cmd.Context(), txID)
			if err != nil {
				return err
			}
			return printJSON(cmd, tx)
		},
	}
	return cmd
}
