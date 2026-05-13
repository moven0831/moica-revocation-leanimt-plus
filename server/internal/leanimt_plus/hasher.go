package leanimt_plus

import (
	"math/big"

	poseidon "github.com/zkmopro/go-poseidon-p256"
)

type Hasher interface {
	Hash2(a, b *big.Int) *big.Int
}

type PoseidonHasher struct{}

func NewPoseidonHasher() *PoseidonHasher {
	return &PoseidonHasher{}
}

func (h *PoseidonHasher) Hash2(a, b *big.Int) *big.Int {
	return poseidon.Hash2(a, b)
}
