package relayerauth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/polygolem/internal/auth"
	"github.com/TrebuchetDynamics/polygolem/internal/gamma"
	"github.com/TrebuchetDynamics/polygolem/internal/relayer"
)

type MintV2KeyRequest struct {
	PrivateKey string
	GammaURL   string
	RelayerURL string
	Now        func() time.Time
}

func MintV2Key(ctx context.Context, req MintV2KeyRequest) (relayer.V2APIKey, error) {
	signer, err := auth.NewPrivateKeySigner(req.PrivateKey, 137)
	if err != nil {
		return relayer.V2APIKey{}, fmt.Errorf("init signer: %w", err)
	}
	now := time.Now
	if req.Now != nil {
		now = req.Now
	}
	session, err := auth.NewSIWESession(signer, req.GammaURL)
	if err != nil {
		return relayer.V2APIKey{}, fmt.Errorf("new siwe session: %w", err)
	}
	if err := session.Login(ctx); err != nil {
		return relayer.V2APIKey{}, fmt.Errorf("siwe login: %w", err)
	}
	maker, err := auth.MakerAddressForSignatureType(signer.Address(), 137, 3)
	if err != nil {
		return relayer.V2APIKey{}, fmt.Errorf("derive deposit wallet maker: %w", err)
	}
	body := gamma.NewCreateProfileRequest(
		signer.Address(),
		maker,
		"metamask",
		now().UnixMilli(),
	)
	if _, err := gamma.CreateProfile(ctx, session.HTTPClient(), req.GammaURL, body); err != nil && !strings.Contains(err.Error(), "HTTP 409") {
		return relayer.V2APIKey{}, fmt.Errorf("create profile: %w", err)
	}
	key, err := relayer.MintV2APIKey(ctx, session.HTTPClient(), req.RelayerURL)
	if err != nil {
		return relayer.V2APIKey{}, fmt.Errorf("mint v2 relayer key: %w", err)
	}
	return key, nil
}
