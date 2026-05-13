package leanimt_plus

import "math/big"

func (t *LeanIMTPlus) GenerateProof(value *big.Int) (*Proof, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if value.Sign() == 0 {
		return nil, ErrInsertZero
	}
	if len(t.leaves) == 0 {
		return nil, ErrEmptyTree
	}

	if idx := t.indexOfLocked(value); idx != -1 {
		return t.buildProofLocked(ProofMembership, value, idx), nil
	}
	lowIdx, err := t.findLowLeafIndexLocked(value)
	if err != nil {
		return nil, err
	}
	return t.buildProofLocked(ProofNonMembership, value, lowIdx), nil
}

// buildProofLocked walks the Merkle path collecting siblings. LeafIndex bits
// are emitted LSB-first, but only for levels where a sibling actually exists
// (unpaired right children are promoted unchanged and contribute no sibling).
//
// Worked example for physIdx = 5 at depth 3:
//
//	level 0: i=5, isRight=1, sibling=nodes[0][4]  → emit bit 1 at bitPos 0
//	level 1: i=2, isRight=0, sibling=nodes[1][3]  → emit bit 0 at bitPos 1
//	level 2: i=1, isRight=1, sibling=nodes[2][0]  → emit bit 1 at bitPos 2
//	LeafIndex = 0b101 = 5
//
// If a level lacks a sibling, neither path bit nor sibling is emitted, and
// bitPos does not advance for that level.
func (t *LeanIMTPlus) buildProofLocked(pt ProofType, value *big.Int, physIdx int) *Proof {
	leaf := t.leaves[physIdx].Clone()
	var siblings []*big.Int
	var leafIndex uint64
	var bitPos uint = 0
	i := physIdx
	depth := len(t.nodes) - 1

	for level := 0; level < depth; level++ {
		isRight := i & 1
		var sibIdx int
		if isRight == 1 {
			sibIdx = i - 1
		} else {
			sibIdx = i + 1
		}
		if sibIdx < len(t.nodes[level]) && t.nodes[level][sibIdx] != nil {
			siblings = append(siblings, new(big.Int).Set(t.nodes[level][sibIdx]))
			if isRight == 1 {
				leafIndex |= 1 << bitPos
			}
			bitPos++
		}
		i >>= 1
	}

	return &Proof{
		ProofType: pt,
		Root:      t.rootLocked(),
		Value:     new(big.Int).Set(value),
		Leaf:      leaf,
		LeafIndex: leafIndex,
		Siblings:  siblings,
	}
}
