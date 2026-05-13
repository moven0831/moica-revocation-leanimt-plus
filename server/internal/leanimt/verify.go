package leanimt

import "math/big"

func VerifyProof(h Hasher, p *Proof) bool {
	if p == nil || p.Root == nil || p.Value == nil || p.Leaf.Value == nil || p.Leaf.NextValue == nil {
		return false
	}

	switch p.ProofType {
	case ProofMembership:
		if p.Leaf.Value.Cmp(p.Value) != 0 {
			return false
		}
	case ProofNonMembership:
		if p.Leaf.Value.Cmp(p.Value) >= 0 {
			return false
		}
		if p.Leaf.NextValue.Sign() != 0 && p.Leaf.NextValue.Cmp(p.Value) <= 0 {
			return false
		}
	default:
		return false
	}

	commitment := h.Hash2(p.Leaf.Value, p.Leaf.NextValue)
	got := walkPath(h, commitment, p.LeafIndex, p.Siblings)
	return got.Cmp(p.Root) == 0
}

func walkPath(h Hasher, leaf *big.Int, leafIndex uint64, siblings []*big.Int) *big.Int {
	node := new(big.Int).Set(leaf)
	for i := 0; i < len(siblings); i++ {
		if (leafIndex>>uint(i))&1 == 1 {
			node = h.Hash2(siblings[i], node)
		} else {
			node = h.Hash2(node, siblings[i])
		}
	}
	return node
}
