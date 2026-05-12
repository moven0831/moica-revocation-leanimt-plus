package leanimt

import (
	"errors"
	"math/big"
	"sync"
)

var (
	ErrEmptyTree           = errors.New("tree is empty")
	ErrInsertZero          = errors.New("cannot insert zero value")
	ErrDuplicate           = errors.New("value already exists in the tree")
	ErrNoValues            = errors.New("no values to insert")
	ErrUnsorted            = errors.New("input is not strictly ascending")
	ErrLowLeafNotFound     = errors.New("invariant violated: no low leaf for value")
	ErrStateLengthMismatch = errors.New("leaves and nodes[0] length disagree")
	ErrEmptyNodes          = errors.New("nodes must have at least one level")
)

// LeanIMTPlus stores the LeanIMT levels and indexed-leaf records side by side.
// leaves[i] is paired with nodes[0][i]. leaves[0] is the sentinel {0,
// smallest-user-value} created on first insert; it is never reported as a
// member and zero is reserved for it. Size() returns user-leaf count
// (excludes sentinel); LeafCount() includes it.
type LeanIMTPlus struct {
	mu     sync.RWMutex
	hash   Hasher
	nodes  [][]*big.Int
	leaves []IndexedLeaf
}

func New(h Hasher) *LeanIMTPlus {
	return &LeanIMTPlus{
		hash:  h,
		nodes: [][]*big.Int{{}},
	}
}

func (t *LeanIMTPlus) Root() *big.Int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.rootLocked()
}

func (t *LeanIMTPlus) rootLocked() *big.Int {
	if len(t.leaves) == 0 {
		return nil
	}
	top := t.nodes[len(t.nodes)-1]
	if len(top) == 0 || top[0] == nil {
		return nil
	}
	return new(big.Int).Set(top[0])
}

func (t *LeanIMTPlus) Depth() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.nodes) - 1
}

func (t *LeanIMTPlus) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if len(t.leaves) == 0 {
		return 0
	}
	return len(t.leaves) - 1
}

func (t *LeanIMTPlus) LeafCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.leaves)
}

func (t *LeanIMTPlus) IndexOf(v *big.Int) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.indexOfLocked(v)
}

func (t *LeanIMTPlus) indexOfLocked(v *big.Int) int {
	if v.Sign() == 0 {
		return -1
	}
	for i := 1; i < len(t.leaves); i++ {
		if t.leaves[i].Value.Cmp(v) == 0 {
			return i
		}
	}
	return -1
}

func (t *LeanIMTPlus) Has(v *big.Int) bool {
	return t.IndexOf(v) != -1
}

// Leaves returns a deep copy excluding the sentinel.
func (t *LeanIMTPlus) Leaves() []IndexedLeaf {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if len(t.leaves) == 0 {
		return nil
	}
	out := make([]IndexedLeaf, 0, len(t.leaves)-1)
	for i := 1; i < len(t.leaves); i++ {
		out = append(out, t.leaves[i].Clone())
	}
	return out
}

// ExportState returns deep copies safe for the caller to mutate.
func (t *LeanIMTPlus) ExportState() ([][]*big.Int, []IndexedLeaf) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	nodes := make([][]*big.Int, len(t.nodes))
	for lvl, level := range t.nodes {
		nodes[lvl] = make([]*big.Int, len(level))
		for i, n := range level {
			if n != nil {
				nodes[lvl][i] = new(big.Int).Set(n)
			}
		}
	}
	leaves := make([]IndexedLeaf, len(t.leaves))
	for i, l := range t.leaves {
		leaves[i] = l.Clone()
	}
	return nodes, leaves
}

// ImportState replaces tree state from a snapshot. Integrity (hash chain,
// sort order) is the caller's responsibility; we only sanity-check shape.
func (t *LeanIMTPlus) ImportState(nodes [][]*big.Int, leaves []IndexedLeaf) error {
	if len(nodes) == 0 {
		return ErrEmptyNodes
	}
	if len(nodes[0]) != len(leaves) {
		return ErrStateLengthMismatch
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	t.nodes = make([][]*big.Int, len(nodes))
	for lvl, level := range nodes {
		t.nodes[lvl] = make([]*big.Int, len(level))
		for i, n := range level {
			if n != nil {
				t.nodes[lvl][i] = new(big.Int).Set(n)
			}
		}
	}
	t.leaves = make([]IndexedLeaf, len(leaves))
	for i, l := range leaves {
		t.leaves[i] = l.Clone()
	}
	return nil
}
