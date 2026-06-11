package bridge

import (
	"context"
	"errors"
	"testing"
)

func TestBuildWithdrawDryRunValidatesAndWarns(t *testing.T) {
	dryRun, err := BuildWithdrawDryRun(validWithdrawRequest())
	if err != nil {
		t.Fatal(err)
	}
	if dryRun.ReadyToSubmit {
		t.Fatal("withdraw dry-run must not be ready to submit")
	}
	if !dryRun.Unsupported {
		t.Fatal("withdraw dry-run must mark live submit unsupported")
	}
	if len(dryRun.SafetyWarnings) == 0 {
		t.Fatal("expected safety warnings")
	}
	if dryRun.Request.RecipientAddress != "0xrecipient" {
		t.Fatalf("request not preserved: %+v", dryRun.Request)
	}
}

func TestBuildWithdrawDryRunRequiresFields(t *testing.T) {
	req := validWithdrawRequest()
	req.RecipientAddress = ""
	_, err := BuildWithdrawDryRun(req)
	if err == nil {
		t.Fatal("expected missing recipient validation error")
	}
}

func TestBuildWithdrawDryRunRejectsNonPositiveAmount(t *testing.T) {
	for _, amount := range []string{"0", "-1", "1.5", "not-a-number"} {
		req := validWithdrawRequest()
		req.FromAmountBaseUnit = amount
		_, err := BuildWithdrawDryRun(req)
		if err == nil {
			t.Fatalf("expected amount validation error for %q", amount)
		}
	}
}

func TestWithdrawReturnsUnsupportedWithoutSubmitting(t *testing.T) {
	client := NewClient("", nil)
	_, err := client.Withdraw(context.Background(), validWithdrawRequest())
	if !errors.Is(err, ErrWithdrawSubmitUnsupported) {
		t.Fatalf("err=%v want ErrWithdrawSubmitUnsupported", err)
	}
}

func validWithdrawRequest() WithdrawRequest {
	return WithdrawRequest{
		FromChainID:        "137",
		FromTokenAddress:   "0xpolymarket-token",
		FromAmountBaseUnit: "1000000",
		ToChainID:          "1",
		ToTokenAddress:     "0xmainnet-token",
		RecipientAddress:   "0xrecipient",
		QuoteID:            "quote-1",
	}
}
