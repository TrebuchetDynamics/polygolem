package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

func relayerCmd(jsonOut bool) *cobra.Command {
	cmd := commandGroup("relayer", "Inspect Polymarket relayer state")
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
