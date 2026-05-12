package crl

import (
	"math/big"
	"sort"
)

// DedupAndSortSerials prepares CRL serials for leanimt.InsertManySorted's
// O(1)-tail fast path. Zero and nil are dropped; the tree reserves zero for
// the sentinel. The returned slice shares element pointers with the input;
// the tree copies values defensively.
func DedupAndSortSerials(serials []*big.Int) []*big.Int {
	if len(serials) == 0 {
		return nil
	}
	out := make([]*big.Int, 0, len(serials))
	for _, s := range serials {
		if s != nil && s.Sign() > 0 {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Cmp(out[j]) < 0 })
	if len(out) <= 1 {
		return out
	}
	w := 1
	for i := 1; i < len(out); i++ {
		if out[i].Cmp(out[w-1]) != 0 {
			out[w] = out[i]
			w++
		}
	}
	return out[:w]
}
