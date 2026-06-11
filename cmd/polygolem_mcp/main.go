package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/TrebuchetDynamics/polygolem/pkg/mcp"
)

func main() {
	server := mcp.NewServer()
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req mcp.Request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = encoder.Encode(mcp.Response{JSONRPC: "2.0", Error: &mcp.ResponseError{Code: -32700, Message: "parse error"}})
			continue
		}
		_ = encoder.Encode(server.Handle(context.Background(), req))
	}
	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "polygolem-mcp stdin: %v\n", err)
		os.Exit(1)
	}
}
