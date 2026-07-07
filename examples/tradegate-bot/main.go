package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/TrebuchetDynamics/polygolem/pkg/clob"
)

type envGate struct{ halted bool }

func (g envGate) CanProceed() bool { return !g.halted }

func main() {
	client := clob.NewClient(clob.Config{TradeGate: envGate{halted: true}})
	_, err := client.CreateLimitOrder(context.Background(), "not-used-when-gate-halts", clob.CreateOrderParams{
		TokenID: "example-token-id",
		Side:    "buy",
		Price:   "0.01",
		Size:    "1",
	})
	if !errors.Is(err, clob.ErrTradingHalted) {
		log.Fatalf("expected trade gate halt, got %v", err)
	}
	fmt.Println("trade gate blocked order before signing or submission")
}
