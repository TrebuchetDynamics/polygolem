// onboard_deposit_wallet is a one-shot, disposable onboarding tool (delete
// after use, like live_siwe_probe): SIWE login with SIGNER_PRIVATE_KEY (env),
// mint a V2 relayer API key, append RELAYER_API_KEY(+_ADDRESS) to the env
// file given as argv[1] (skipped if already present), then run the standard
// OnboardDepositWallet flow: deploy the deposit wallet if needed and submit
// the pUSD+CTF approval batch. Prints only non-secret status fields.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	internalauth "github.com/TrebuchetDynamics/polygolem/internal/auth"
	internalrelayer "github.com/TrebuchetDynamics/polygolem/internal/relayer"
	"github.com/TrebuchetDynamics/polygolem/pkg/relayer"
)

const (
	gammaURL   = "https://gamma-api.polymarket.com"
	relayerURL = "https://relayer-v2.polymarket.com"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: onboard_deposit_wallet <env-file> [--skip-deploy]")
	}
	envFile := os.Args[1]
	skipDeploy := len(os.Args) > 2 && os.Args[2] == "--skip-deploy"
	statusOnly := len(os.Args) > 2 && os.Args[2] == "--status"
	pk := strings.TrimSpace(os.Getenv("SIGNER_PRIVATE_KEY"))
	if pk == "" {
		return fmt.Errorf("SIGNER_PRIVATE_KEY not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	signer, err := internalauth.NewPrivateKeySigner(pk, 137)
	if err != nil {
		return fmt.Errorf("init signer: %w", err)
	}
	fmt.Println("owner EOA:", signer.Address())

	session, err := internalauth.NewSIWESession(signer, gammaURL)
	if err != nil {
		return fmt.Errorf("siwe session: %w", err)
	}
	if err := session.Login(ctx); err != nil {
		return fmt.Errorf("siwe login: %w", err)
	}
	fmt.Println("siwe login: ok")

	v2Key, err := internalrelayer.MintV2APIKey(ctx, session.HTTPClient(), relayerURL)
	if err != nil {
		return fmt.Errorf("mint relayer key: %w", err)
	}
	fmt.Println("relayer v2 key minted for address:", v2Key.Address)

	existing, err := os.ReadFile(envFile)
	if err != nil {
		return fmt.Errorf("read env file: %w", err)
	}
	if strings.Contains(string(existing), "RELAYER_API_KEY=") {
		fmt.Println("env file already has RELAYER_API_KEY; not appending")
	} else {
		f, err := os.OpenFile(envFile, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("open env file: %w", err)
		}
		_, werr := fmt.Fprintf(f, "RELAYER_API_KEY=%s\nRELAYER_API_KEY_ADDRESS=%s\n", v2Key.Key, v2Key.Address)
		cerr := f.Close()
		if werr != nil {
			return fmt.Errorf("append relayer key: %w", werr)
		}
		if cerr != nil {
			return fmt.Errorf("close env file: %w", cerr)
		}
		fmt.Println("relayer key appended to env file")
	}

	client, err := relayer.NewV2(relayerURL, relayer.V2APIKey{Key: v2Key.Key, Address: v2Key.Address}, 137)
	if err != nil {
		return fmt.Errorf("relayer client: %w", err)
	}

	// --batch <wallet>: submit the approval batch against an explicit wallet
	// address (bypasses local derivation — used when the factory-reported
	// wallet differs from the SDK-derived one).
	if len(os.Args) > 3 && os.Args[2] == "--batch" {
		actualWallet := strings.TrimSpace(os.Args[3])
		calls := append(relayer.BuildApprovalCalls(), relayer.BuildAdapterApprovalCalls()...)
		nonce, err := client.GetNonce(ctx, signer.Address())
		if err != nil {
			return fmt.Errorf("nonce: %w", err)
		}
		deadline := relayer.BuildDeadline(0)
		sig, err := internalrelayer.SignWalletBatch(signer, actualWallet, nonce, deadline, calls)
		if err != nil {
			return fmt.Errorf("sign batch: %w", err)
		}
		tx, err := client.SubmitWalletBatch(ctx, signer.Address(), actualWallet, nonce, sig, deadline, calls)
		if err != nil {
			return fmt.Errorf("submit batch: %w", err)
		}
		fmt.Printf("batch submitted: id=%s state=%s\n", tx.TransactionID, tx.State)
		final, err := client.PollTransaction(ctx, tx.TransactionID, 50, 2*time.Second)
		if err != nil {
			return fmt.Errorf("poll batch: %w", err)
		}
		fmt.Printf("batch final: state=%s hash=%s\n", final.State, final.TransactionHash)
		return nil
	}

	if statusOnly {
		owner, wallet, err := relayer.DepositWalletAddress(pk)
		if err != nil {
			return fmt.Errorf("derive: %w", err)
		}
		fmt.Println("derived owner:", owner, "deposit wallet:", wallet)
		deployed, err := client.IsDeployed(ctx, owner)
		fmt.Printf("relayer IsDeployed(owner): %v err=%v\n", deployed, err)
		nonce, err := client.GetNonce(ctx, owner)
		fmt.Printf("relayer GetNonce(owner): %q err=%v\n", nonce, err)
		return nil
	}

	result, err := relayer.OnboardDepositWallet(ctx, client, pk, relayer.OnboardOptions{SkipDeploy: skipDeploy})
	if err != nil {
		return fmt.Errorf("onboard: %w", err)
	}
	fmt.Println("deposit wallet:", result.DepositWallet)
	if result.Deploy != nil {
		fmt.Printf("deploy: state=%s already_deployed=%v skipped=%v tx=%s\n",
			result.Deploy.State, result.Deploy.AlreadyDeployed, result.Deploy.Skipped, result.Deploy.TransactionID)
	}
	if result.Approve != nil {
		fmt.Printf("approvals: state=%s calls=%d skipped=%v tx=%s\n",
			result.Approve.State, result.Approve.CallCount, result.Approve.Skipped, result.Approve.TransactionID)
	}
	fmt.Println("onboarding complete")
	return nil
}
