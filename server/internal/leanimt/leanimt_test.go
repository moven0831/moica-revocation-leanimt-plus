package leanimt

import (
	"math/big"
	"math/rand"
	"sort"
	"testing"
)

func bigsFromInts(vs ...int64) []*big.Int {
	out := make([]*big.Int, len(vs))
	for i, v := range vs {
		out[i] = big.NewInt(v)
	}
	return out
}

func TestEmptyTree(t *testing.T) {
	tree := New(NewPoseidonHasher())
	if tree.Size() != 0 {
		t.Fatalf("empty Size: got %d", tree.Size())
	}
	if tree.LeafCount() != 0 {
		t.Fatalf("empty LeafCount: got %d", tree.LeafCount())
	}
	if tree.Depth() != 0 {
		t.Fatalf("empty Depth: got %d", tree.Depth())
	}
	if tree.Root() != nil {
		t.Fatalf("empty Root: got %v", tree.Root())
	}
	if _, err := tree.GenerateProof(big.NewInt(1)); err != ErrEmptyTree {
		t.Fatalf("GenerateProof on empty: got %v", err)
	}
}

func TestFirstInsertCreatesSentinel(t *testing.T) {
	tree := New(NewPoseidonHasher())
	if err := tree.Insert(big.NewInt(42)); err != nil {
		t.Fatal(err)
	}
	if tree.Size() != 1 {
		t.Fatalf("Size: got %d want 1", tree.Size())
	}
	if tree.LeafCount() != 2 {
		t.Fatalf("LeafCount: got %d want 2", tree.LeafCount())
	}
	if tree.Depth() != 1 {
		t.Fatalf("Depth: got %d want 1", tree.Depth())
	}
	if !tree.Has(big.NewInt(42)) {
		t.Fatal("Has(42): false")
	}
	if tree.Has(big.NewInt(0)) {
		t.Fatal("Has(0): true (sentinel must not be reported)")
	}
}

func TestInsertZeroRejected(t *testing.T) {
	tree := New(NewPoseidonHasher())
	if err := tree.Insert(big.NewInt(0)); err != ErrInsertZero {
		t.Fatalf("Insert(0): got %v want %v", err, ErrInsertZero)
	}
}

func TestInsertDuplicateRejected(t *testing.T) {
	tree := New(NewPoseidonHasher())
	if err := tree.Insert(big.NewInt(10)); err != nil {
		t.Fatal(err)
	}
	if err := tree.Insert(big.NewInt(10)); err != ErrDuplicate {
		t.Fatalf("Insert duplicate: got %v want %v", err, ErrDuplicate)
	}
}

func TestInsertMaintainsSortedLinkedList(t *testing.T) {
	tree := New(NewPoseidonHasher())
	for _, v := range []int64{50, 10, 30, 70, 20, 60, 40} {
		if err := tree.Insert(big.NewInt(v)); err != nil {
			t.Fatal(err)
		}
	}

	leaves := append([]IndexedLeaf{{Value: big.NewInt(0), NextValue: nil}}, tree.Leaves()...)
	leaves[0].NextValue = lookupNext(tree, big.NewInt(0))

	expected := []int64{0, 10, 20, 30, 40, 50, 60, 70}
	cur := big.NewInt(0)
	for i, want := range expected {
		if cur.Cmp(big.NewInt(want)) != 0 {
			t.Fatalf("walk[%d]: got %s want %d", i, cur.Text(10), want)
		}
		next := lookupNext(tree, cur)
		if next == nil {
			if i != len(expected)-1 {
				t.Fatalf("walk[%d]: unexpected tail", i)
			}
			break
		}
		cur = next
	}
}

// lookupNext finds the leaf whose Value == v and returns its NextValue, or nil
// if the sequence terminates. The sentinel is checked first.
func lookupNext(tree *LeanIMTPlus, v *big.Int) *big.Int {
	tree.mu.RLock()
	defer tree.mu.RUnlock()
	for _, l := range tree.leaves {
		if l.Value.Cmp(v) == 0 {
			if l.NextValue.Sign() == 0 {
				return nil
			}
			return new(big.Int).Set(l.NextValue)
		}
	}
	return nil
}

func TestMembershipProofRoundTrip(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)
	values := bigsFromInts(7, 19, 3, 88, 42, 256, 1, 1000, 33)
	for _, v := range values {
		if err := tree.Insert(v); err != nil {
			t.Fatal(err)
		}
	}
	for _, v := range values {
		p, err := tree.GenerateProof(v)
		if err != nil {
			t.Fatal(err)
		}
		if p.ProofType != ProofMembership {
			t.Errorf("value %s: ProofType=%d want %d", v, p.ProofType, ProofMembership)
		}
		if p.Leaf.Value.Cmp(v) != 0 {
			t.Errorf("value %s: Leaf.Value=%s", v, p.Leaf.Value)
		}
		if !VerifyProof(h, p) {
			t.Errorf("value %s: VerifyProof false", v)
		}
	}
}

func TestNonMembershipProof_ThreeCases(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)
	for _, v := range []int64{3, 18, 25} {
		if err := tree.Insert(big.NewInt(v)); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		name     string
		value    int64
		lowValue int64
		lowNext  int64
	}{
		{"sentinel-as-low", 1, 0, 3},
		{"strictly-between", 20, 18, 25},
		{"tail-as-low", 100, 25, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := tree.GenerateProof(big.NewInt(tc.value))
			if err != nil {
				t.Fatal(err)
			}
			if p.ProofType != ProofNonMembership {
				t.Fatalf("ProofType=%d", p.ProofType)
			}
			if p.Leaf.Value.Int64() != tc.lowValue {
				t.Fatalf("Leaf.Value=%s want %d", p.Leaf.Value, tc.lowValue)
			}
			if p.Leaf.NextValue.Int64() != tc.lowNext {
				t.Fatalf("Leaf.NextValue=%s want %d", p.Leaf.NextValue, tc.lowNext)
			}
			if !VerifyProof(h, p) {
				t.Fatal("VerifyProof false")
			}
		})
	}
}

func TestTamperedProofRejected(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)
	for _, v := range []int64{5, 11, 17, 23, 29, 31} {
		if err := tree.Insert(big.NewInt(v)); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("forged-membership-value", func(t *testing.T) {
		p, err := tree.GenerateProof(big.NewInt(11))
		if err != nil {
			t.Fatal(err)
		}
		p.Value = big.NewInt(13)
		if VerifyProof(h, p) {
			t.Fatal("accepted forged value")
		}
	})

	t.Run("flipped-proof-type", func(t *testing.T) {
		p, err := tree.GenerateProof(big.NewInt(11))
		if err != nil {
			t.Fatal(err)
		}
		p.ProofType = ProofNonMembership
		if VerifyProof(h, p) {
			t.Fatal("accepted flipped type")
		}
	})

	t.Run("mutated-sibling", func(t *testing.T) {
		p, err := tree.GenerateProof(big.NewInt(11))
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Siblings) == 0 {
			t.Skip("no siblings to mutate")
		}
		p.Siblings[0] = new(big.Int).Add(p.Siblings[0], big.NewInt(1))
		if VerifyProof(h, p) {
			t.Fatal("accepted mutated sibling")
		}
	})

	t.Run("forged-non-membership-low-leaf", func(t *testing.T) {
		p, err := tree.GenerateProof(big.NewInt(20))
		if err != nil {
			t.Fatal(err)
		}
		p.Leaf.Value = big.NewInt(19)
		if VerifyProof(h, p) {
			t.Fatal("accepted forged low leaf")
		}
	})
}

func TestInsertManySorted_MatchesIndividualInserts(t *testing.T) {
	h := NewPoseidonHasher()
	values := bigsFromInts(2, 7, 11, 13, 19, 23, 41, 67, 89)

	a := New(h)
	for _, v := range values {
		if err := a.Insert(v); err != nil {
			t.Fatal(err)
		}
	}
	b := New(h)
	if err := b.InsertManySorted(values); err != nil {
		t.Fatal(err)
	}

	if a.Root().Cmp(b.Root()) != 0 {
		t.Fatalf("roots differ: a=%s b=%s", a.Root(), b.Root())
	}
	if a.Size() != b.Size() || a.Depth() != b.Depth() {
		t.Fatalf("shape differs")
	}
}

func TestInsertManySorted_UnsortedInputRejected(t *testing.T) {
	tree := New(NewPoseidonHasher())
	err := tree.InsertManySorted(bigsFromInts(10, 5))
	if err != ErrUnsorted {
		t.Fatalf("got %v want %v", err, ErrUnsorted)
	}
}

func TestInsertManySorted_DuplicateRejected(t *testing.T) {
	tree := New(NewPoseidonHasher())
	err := tree.InsertManySorted(bigsFromInts(3, 3))
	if err != ErrDuplicate {
		t.Fatalf("got %v want %v", err, ErrDuplicate)
	}
}

func TestSelfConsistency_100RandomValues(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)
	r := rand.New(rand.NewSource(0xC0FFEE))
	seen := make(map[int64]struct{}, 100)
	values := make([]*big.Int, 0, 100)
	for len(values) < 100 {
		v := r.Int63n(1<<40) + 1
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		values = append(values, big.NewInt(v))
	}
	sortBigs(values)
	if err := tree.InsertManySorted(values); err != nil {
		t.Fatal(err)
	}

	for _, v := range values {
		p, err := tree.GenerateProof(v)
		if err != nil {
			t.Fatalf("GenerateProof(%s): %v", v, err)
		}
		if !VerifyProof(h, p) {
			t.Fatalf("VerifyProof failed for member %s", v)
		}
	}

	for i := 0; i < 100; i++ {
		var v *big.Int
		for {
			cand := big.NewInt(r.Int63n(1<<40) + 1)
			if _, dup := seen[cand.Int64()]; !dup {
				v = cand
				break
			}
		}
		p, err := tree.GenerateProof(v)
		if err != nil {
			t.Fatalf("GenerateProof(%s): %v", v, err)
		}
		if p.ProofType != ProofNonMembership {
			t.Fatalf("ProofType mismatch for %s", v)
		}
		if !VerifyProof(h, p) {
			t.Fatalf("VerifyProof failed for non-member %s", v)
		}
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	h := NewPoseidonHasher()
	a := New(h)
	for _, v := range []int64{5, 12, 23, 47, 88, 100} {
		if err := a.Insert(big.NewInt(v)); err != nil {
			t.Fatal(err)
		}
	}

	nodes, leaves := a.ExportState()
	b := New(h)
	if err := b.ImportState(nodes, leaves); err != nil {
		t.Fatal(err)
	}

	if a.Root().Cmp(b.Root()) != 0 {
		t.Fatalf("roots differ after import")
	}
	for _, v := range []int64{5, 100, 23} {
		pa, _ := a.GenerateProof(big.NewInt(v))
		pb, _ := b.GenerateProof(big.NewInt(v))
		if pa.Root.Cmp(pb.Root) != 0 || pa.LeafIndex != pb.LeafIndex || len(pa.Siblings) != len(pb.Siblings) {
			t.Fatalf("proof for %d differs", v)
		}
	}
}

func TestInsertManySorted_ConsecutiveBatches(t *testing.T) {
	h := NewPoseidonHasher()
	all := bigsFromInts(2, 5, 9, 14, 21, 28, 35, 42, 50, 67, 89, 101, 200)

	single := New(h)
	if err := single.InsertManySorted(all); err != nil {
		t.Fatal(err)
	}

	batched := New(h)
	if err := batched.InsertManySorted(all[:5]); err != nil {
		t.Fatal(err)
	}
	if err := batched.InsertManySorted(all[5:9]); err != nil {
		t.Fatal(err)
	}
	if err := batched.InsertManySorted(all[9:]); err != nil {
		t.Fatal(err)
	}

	if single.Root().Cmp(batched.Root()) != 0 {
		t.Fatalf("consecutive batches diverged: single=%s batched=%s",
			single.Root().Text(16), batched.Root().Text(16))
	}
	if single.Size() != batched.Size() || single.Depth() != batched.Depth() {
		t.Fatalf("shape diverged: single=(%d,%d) batched=(%d,%d)",
			single.Size(), single.Depth(), batched.Size(), batched.Depth())
	}
	for _, v := range all {
		p, err := batched.GenerateProof(v)
		if err != nil {
			t.Fatal(err)
		}
		if !VerifyProof(h, p) {
			t.Errorf("VerifyProof failed for %s after consecutive batches", v)
		}
	}
}

func TestInsertManySorted_AfterIndividualInsert(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)
	if err := tree.Insert(big.NewInt(10)); err != nil {
		t.Fatal(err)
	}
	if err := tree.Insert(big.NewInt(30)); err != nil {
		t.Fatal(err)
	}
	if err := tree.InsertManySorted(bigsFromInts(50, 70, 90)); err != nil {
		t.Fatalf("sorted batch after individual inserts: %v", err)
	}
	for _, v := range []int64{10, 30, 50, 70, 90} {
		p, err := tree.GenerateProof(big.NewInt(v))
		if err != nil {
			t.Fatal(err)
		}
		if !VerifyProof(h, p) {
			t.Errorf("VerifyProof failed for %d after mixed paths", v)
		}
	}
}

func TestLargeSortedInsertRootIsStable(t *testing.T) {
	h := NewPoseidonHasher()
	values := bigsFromInts(3, 7, 10, 18, 25, 41)
	a := New(h)
	if err := a.InsertManySorted(values); err != nil {
		t.Fatal(err)
	}
	b := New(h)
	if err := b.InsertManySorted(values); err != nil {
		t.Fatal(err)
	}
	if a.Root().Cmp(b.Root()) != 0 {
		t.Fatalf("roots not deterministic")
	}
	p, err := a.GenerateProof(big.NewInt(20))
	if err != nil {
		t.Fatal(err)
	}
	if p.ProofType != ProofNonMembership {
		t.Fatalf("ProofType=%d", p.ProofType)
	}
	if !VerifyProof(h, p) {
		t.Fatal("VerifyProof failed")
	}
}

func sortBigs(v []*big.Int) {
	sort.Slice(v, func(i, j int) bool { return v[i].Cmp(v[j]) < 0 })
}
