package leanimt

import "math/big"

// IndexedLeaf carries a (value, nextValue) pair where NextValue points to the
// next-larger user value or zero at the tail. The implicit sorted linked list
// across all leaves is what makes non-membership proofs possible: for any v
// outside the tree, exactly one leaf satisfies value < v < nextValue.
type IndexedLeaf struct {
	Value     *big.Int
	NextValue *big.Int
}

func (l IndexedLeaf) Clone() IndexedLeaf {
	return IndexedLeaf{
		Value:     new(big.Int).Set(l.Value),
		NextValue: new(big.Int).Set(l.NextValue),
	}
}
