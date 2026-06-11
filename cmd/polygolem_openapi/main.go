package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/TrebuchetDynamics/polygolem/pkg/openapi"
)

func main() {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(openapi.Spec()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "polygolem-openapi: %v\n", err)
		os.Exit(1)
	}
}
