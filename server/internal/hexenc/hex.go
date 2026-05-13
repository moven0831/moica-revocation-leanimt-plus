// Package hexenc centralizes the "0x"-prefixed big.Int hex encoding used by
// every API surface (REST, gRPC, JSON snapshot, WASM, on-chain root posting).
package hexenc

import (
	"fmt"
	"math/big"
	"strings"
)

// Encode formats n as "0x<lowercase-hex>". Nil and zero both encode to "0x0".
func Encode(n *big.Int) string {
	if n == nil || n.Sign() == 0 {
		return "0x0"
	}
	return "0x" + n.Text(16)
}

// EncodeSlice maps Encode over a slice.
func EncodeSlice(ns []*big.Int) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		out[i] = Encode(n)
	}
	return out
}

// Decode parses an "0x"-prefixed (or bare) hex string. "0x0" and "0" both
// decode to a fresh zero. Empty input is rejected; the snapshot loader uses
// a separate sentinel for the "absent node" case.
func Decode(s string) (*big.Int, error) {
	stripped := strings.TrimPrefix(s, "0x")
	if stripped == "" {
		return nil, fmt.Errorf("empty hex string")
	}
	if stripped == "0" {
		return new(big.Int), nil
	}
	n, ok := new(big.Int).SetString(stripped, 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex: %q", s)
	}
	return n, nil
}
