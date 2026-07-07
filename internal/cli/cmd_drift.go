package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/TrebuchetDynamics/polygolem/pkg/upstreamdrift"
	"github.com/spf13/cobra"
)

func driftCmd(_ bool) *cobra.Command {
	cmd := commandGroup("check-upstream", "Check read-only upstream Polymarket drift", driftLLMSCmd())
	cmd.Long = `Check whether Polymarket's official docs still advertise the surfaces polygolem
depends on. Read-only, credential-free, and offline: it runs against a saved
llms.txt index, so it is safe in CI or air-gapped review.

  curl -fsSL https://docs.polymarket.com/llms.txt -o /tmp/llms.txt
  polygolem check-upstream llms --file /tmp/llms.txt`
	return cmd
}

func driftLLMSCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "llms",
		Short: "Check a saved docs.polymarket.com llms.txt index",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readLLMSInput(file, cmd.InOrStdin())
			if err != nil {
				return err
			}
			report := upstreamdrift.CheckLLMS(string(body))
			if jsonEnabled(cmd) {
				if err := writeCommandJSON(cmd, report); err != nil {
					return err
				}
			} else {
				writeDriftText(cmd.OutOrStdout(), report)
			}
			if !report.OK {
				return fmt.Errorf("missing official Polymarket docs for %d surface(s)", len(report.Missing))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "saved docs.polymarket.com/llms.txt file; reads stdin when empty")
	return cmd
}

func readLLMSInput(file string, stdin io.Reader) ([]byte, error) {
	if file != "" {
		return os.ReadFile(file)
	}
	return io.ReadAll(stdin)
}

func writeDriftText(w io.Writer, report upstreamdrift.Report) {
	if report.OK {
		_, _ = fmt.Fprintf(w, "upstream drift: ok (%d surfaces checked)\n", len(report.Checked))
		return
	}
	_, _ = fmt.Fprintln(w, "upstream drift: missing official docs")
	for _, surface := range report.Missing {
		_, _ = fmt.Fprintf(w, "- %s (%s): %s\n", surface.ID, surface.Service, surface.Expected)
	}
}
