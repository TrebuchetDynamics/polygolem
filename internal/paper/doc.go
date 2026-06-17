// Package paper holds local-only paper-trading state — cash, positions, and
// fills — used to simulate buys and sells without touching any authenticated
// Polymarket endpoint.
//
// State is in-memory and per-process: each CLI invocation starts from a fresh
// account, and nothing is written to disk. (On-disk persistence across runs is
// a possible future enhancement and is intentionally not implemented yet.)
// Useful for offline development and edge validation.
//
// This package is internal and not part of the polygolem public SDK.
package paper
