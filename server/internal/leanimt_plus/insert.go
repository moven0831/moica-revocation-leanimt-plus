package leanimt_plus

import (
	"math/big"
	"math/bits"
)

var zero = big.NewInt(0)

func (t *LeanIMTPlus) Insert(v *big.Int) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.insertBatchLocked([]*big.Int{v})
}

func (t *LeanIMTPlus) InsertMany(values []*big.Int) error {
	if len(values) == 0 {
		return ErrNoValues
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.insertBatchLocked(values)
}

// InsertManySorted exists alongside the naive InsertMany because the
// reference's `_findLowLeafIndex` is O(n) per insert — O(n²) over a batch,
// which is unworkable at 412k entries. For strictly-ascending input every
// new value is the new tail, so we skip the scan entirely and hash each leaf
// commitment only once with its final NextValue.
func (t *LeanIMTPlus) InsertManySorted(values []*big.Int) error {
	return t.InsertManyWithProgress(values, 0, nil)
}

func (t *LeanIMTPlus) InsertManyWithProgress(values []*big.Int, batchSize int, onProgress func(done, total int)) error {
	if len(values) == 0 {
		return ErrNoValues
	}
	for _, v := range values {
		if v.Sign() == 0 {
			return ErrInsertZero
		}
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	needed := len(t.leaves) + len(values)
	if len(t.leaves) == 0 {
		needed++
	}
	t.reserveLockedLevel0(needed)

	modified := make(map[int]struct{}, needed)
	start := 0

	if len(t.leaves) == 0 {
		first := values[0]
		sentinel := IndexedLeaf{Value: new(big.Int).Set(zero), NextValue: new(big.Int).Set(first)}
		t.leaves = append(t.leaves, sentinel)
		t.nodes[0] = append(t.nodes[0], t.hash.Hash2(sentinel.Value, sentinel.NextValue))
		modified[0] = struct{}{}

		t.leaves = append(t.leaves, IndexedLeaf{Value: new(big.Int).Set(first), NextValue: new(big.Int).Set(zero)})
		t.nodes[0] = append(t.nodes[0], nil)
		modified[1] = struct{}{}
		start = 1
	}

	// Each iteration finalizes the previous tail's hash (we now know its
	// NextValue) and appends a new tail with a nil placeholder. The final
	// tail's hash is filled after the loop. This keeps level-0 hashes at
	// N+1 total instead of 2N+1.
	for i := start; i < len(values); i++ {
		v := values[i]
		tailIdx := len(t.leaves) - 1
		tail := t.leaves[tailIdx]
		if tail.NextValue.Sign() != 0 {
			return ErrUnsorted
		}
		cmp := v.Cmp(tail.Value)
		if cmp == 0 {
			return ErrDuplicate
		}
		if cmp < 0 {
			return ErrUnsorted
		}
		t.leaves[tailIdx] = IndexedLeaf{
			Value:     new(big.Int).Set(tail.Value),
			NextValue: new(big.Int).Set(v),
		}
		t.nodes[0][tailIdx] = t.hash.Hash2(tail.Value, v)
		modified[tailIdx] = struct{}{}

		t.leaves = append(t.leaves, IndexedLeaf{Value: new(big.Int).Set(v), NextValue: new(big.Int).Set(zero)})
		t.nodes[0] = append(t.nodes[0], nil)
		modified[len(t.leaves)-1] = struct{}{}

		if batchSize > 0 && onProgress != nil && (i-start+1)%batchSize == 0 {
			onProgress(i-start+1, len(values)-start)
		}
	}

	lastIdx := len(t.leaves) - 1
	if t.nodes[0][lastIdx] == nil {
		last := t.leaves[lastIdx]
		t.nodes[0][lastIdx] = t.hash.Hash2(last.Value, last.NextValue)
	}

	t.recomputeLocked(modified)

	if onProgress != nil {
		onProgress(len(values)-start, len(values)-start)
	}
	return nil
}

// reserveLockedLevel0 grows t.leaves and t.nodes[0] capacity in one shot so a
// 412k-entry build doesn't pay ~15 reallocations + ~50MB of memcpy.
func (t *LeanIMTPlus) reserveLockedLevel0(targetLen int) {
	if cap(t.leaves) < targetLen {
		newLeaves := make([]IndexedLeaf, len(t.leaves), targetLen)
		copy(newLeaves, t.leaves)
		t.leaves = newLeaves
	}
	if cap(t.nodes[0]) < targetLen {
		newRow := make([]*big.Int, len(t.nodes[0]), targetLen)
		copy(newRow, t.nodes[0])
		t.nodes[0] = newRow
	}
}

func (t *LeanIMTPlus) insertBatchLocked(values []*big.Int) error {
	modified := make(map[int]struct{}, len(values)*2)

	for _, v := range values {
		if v.Sign() == 0 {
			return ErrInsertZero
		}

		if len(t.leaves) == 0 {
			t.appendLeafLocked(IndexedLeaf{Value: new(big.Int).Set(zero), NextValue: new(big.Int).Set(v)}, modified)
			t.appendLeafLocked(IndexedLeaf{Value: new(big.Int).Set(v), NextValue: new(big.Int).Set(zero)}, modified)
			continue
		}

		lowIdx, err := t.findLowLeafIndexLocked(v)
		if err != nil {
			return err
		}
		low := t.leaves[lowIdx]
		if low.NextValue.Sign() != 0 && low.NextValue.Cmp(v) == 0 {
			return ErrDuplicate
		}

		t.appendLeafLocked(IndexedLeaf{Value: new(big.Int).Set(v), NextValue: new(big.Int).Set(low.NextValue)}, modified)
		t.writeLeafLocked(lowIdx, IndexedLeaf{Value: new(big.Int).Set(low.Value), NextValue: new(big.Int).Set(v)})
		modified[lowIdx] = struct{}{}
	}

	t.recomputeLocked(modified)
	return nil
}

func (t *LeanIMTPlus) findLowLeafIndexLocked(v *big.Int) (int, error) {
	for i := 0; i < len(t.leaves); i++ {
		cur := t.leaves[i]
		if cur.Value.Cmp(v) >= 0 {
			continue
		}
		if cur.NextValue.Sign() == 0 || cur.NextValue.Cmp(v) >= 0 {
			return i, nil
		}
	}
	return -1, ErrLowLeafNotFound
}

func (t *LeanIMTPlus) appendLeafLocked(leaf IndexedLeaf, modified map[int]struct{}) {
	idx := len(t.leaves)
	t.leaves = append(t.leaves, leaf)
	t.nodes[0] = append(t.nodes[0], t.hash.Hash2(leaf.Value, leaf.NextValue))
	modified[idx] = struct{}{}
}

func (t *LeanIMTPlus) writeLeafLocked(idx int, leaf IndexedLeaf) {
	t.leaves[idx] = leaf
	t.nodes[0][idx] = t.hash.Hash2(leaf.Value, leaf.NextValue)
}

func (t *LeanIMTPlus) recomputeLocked(modifiedLeaves map[int]struct{}) {
	size := len(t.nodes[0])
	targetDepth := 0
	if size > 1 {
		targetDepth = bits.Len(uint(size - 1))
	}
	for len(t.nodes)-1 < targetDepth {
		t.nodes = append(t.nodes, []*big.Int{})
	}
	if targetDepth == 0 {
		return
	}

	modified := make(map[int]struct{}, len(modifiedLeaves))
	for i := range modifiedLeaves {
		modified[i>>1] = struct{}{}
	}

	for level := 1; level <= targetDepth; level++ {
		next := make(map[int]struct{}, len(modified))
		for idx := range modified {
			var left, right *big.Int
			if 2*idx < len(t.nodes[level-1]) {
				left = t.nodes[level-1][2*idx]
			}
			if 2*idx+1 < len(t.nodes[level-1]) {
				right = t.nodes[level-1][2*idx+1]
			}
			var h *big.Int
			if right != nil {
				h = t.hash.Hash2(left, right)
			} else {
				h = left
			}
			for len(t.nodes[level]) <= idx {
				t.nodes[level] = append(t.nodes[level], nil)
			}
			t.nodes[level][idx] = h
			next[idx>>1] = struct{}{}
		}
		modified = next
	}
}
