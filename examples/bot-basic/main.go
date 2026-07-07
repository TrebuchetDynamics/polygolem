package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/TrebuchetDynamics/polygolem/pkg/universal"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := universal.NewClient(universal.Config{})
	markets, err := client.ActiveMarkets(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("active markets: %d\n", len(markets))
	for i, market := range markets {
		if i >= 5 {
			break
		}
		fmt.Printf("- %s (%s)\n", market.Question, market.Slug)
	}
}
