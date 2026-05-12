package leanimt

import "math/big"

type ProofType uint8

const (
	ProofMembership    ProofType = 0
	ProofNonMembership ProofType = 1
)

// LeafIndex packs path bits LSB-first: bit i selects the direction of
// Siblings[i] during root reconstruction. Bits are only emitted for levels
// where a sibling exists, so LeafIndex is NOT the physical leaf index — at
// unpaired-right levels LeanIMT promotes the left child unchanged and neither
// a sibling nor a path bit is produced.
type Proof struct {
	ProofType ProofType
	Root      *big.Int
	Value     *big.Int
	Leaf      IndexedLeaf
	LeafIndex uint64
	Siblings  []*big.Int
}
