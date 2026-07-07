// redeem_positions is a one-shot, disposable settlement tool (delete after
// use): find the deposit wallet's redeemable positions via data-api and
// submit the redeem batch through the V2 relayer (gasless). Reads
// SIGNER_PRIVATE_KEY + RELAYER_API_KEY(+_ADDRESS) from env. Prints no key
// material.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/data"
	"github.com/TrebuchetDynamics/polygolem/pkg/relayer"
	"github.com/TrebuchetDynamics/polygolem/pkg/settlement"
)

const relayerURL = "https://relayer-v2.polymarket.com"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	pk := strings.TrimSpace(os.Getenv("SIGNER_PRIVATE_KEY"))
	rk := strings.TrimSpace(os.Getenv("RELAYER_API_KEY"))
	ra := strings.TrimSpace(os.Getenv("RELAYER_API_KEY_ADDRESS"))
	if pk == "" || rk == "" || ra == "" {
		return fmt.Errorf("SIGNER_PRIVATE_KEY, RELAYER_API_KEY, RELAYER_API_KEY_ADDRESS required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	owner, wallet, err := relayer.DepositWalletAddress(pk)
	if err != nil {
		return err
	}
	fmt.Println("owner:", owner, "deposit wallet:", wallet)

	dataClient := data.NewClient(data.Config{})
	positions, err := settlement.FindRedeemable(ctx, dataClient, wallet)
	if err != nil {
		return err
	}
	fmt.Println("redeemable positions:", len(positions))
	for _, p := range positions {
		fmt.Printf("  %s %s size=%v\n", p.Title, p.Outcome, p.Size)
	}
	if len(positions) == 0 {
		fmt.Println("nothing to redeem")
		return nil
	}

	rc, err := relayer.NewV2(relayerURL, relayer.V2APIKey{Key: rk, Address: ra}, 137)
	if err != nil {
		return err
	}
	result, err := settlement.SubmitRedeem(ctx, rc, pk, positions, 0)
	if err != nil {
		return err
	}
	fmt.Printf("redeem submitted: tx_id=%s state=%s calls=%d\n", result.TransactionID, result.State, result.CallCount)
	final, err := rc.PollTransaction(ctx, result.TransactionID, 50, 2*time.Second)
	if err != nil {
		return fmt.Errorf("poll redeem: %w", err)
	}
	fmt.Printf("redeem final: state=%s hash=%s\n", final.State, final.TransactionHash)
	return nil
}
