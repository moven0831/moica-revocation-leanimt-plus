package leanimt

import (
	"math/big"
	"math/rand"
	"sort"
	"testing"
)

func generateSortedUniqueBigInts(n int, seed int64) []*big.Int {
	r := rand.New(rand.NewSource(seed))
	seen := make(map[int64]struct{}, n)
	values := make([]*big.Int, 0, n)
	for len(values) < n {
		v := r.Int63n(1<<48) + 1
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		values = append(values, big.NewInt(v))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Cmp(values[j]) < 0 })
	return values
}

func BenchmarkInsertManySorted_1k(b *testing.B) {
	benchInsertManySorted(b, 1_000)
}

func BenchmarkInsertManySorted_10k(b *testing.B) {
	benchInsertManySorted(b, 10_000)
}

func BenchmarkInsertManySorted_100k(b *testing.B) {
	benchInsertManySorted(b, 100_000)
}

func BenchmarkInsertMany_Naive_1k(b *testing.B) {
	benchInsertManyNaive(b, 1_000)
}

func BenchmarkInsertMany_Naive_10k(b *testing.B) {
	benchInsertManyNaive(b, 10_000)
}

func benchInsertManySorted(b *testing.B, n int) {
	values := generateSortedUniqueBigInts(n, int64(n))
	h := NewPoseidonHasher()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree := New(h)
		if err := tree.InsertManySorted(values); err != nil {
			b.Fatal(err)
		}
	}
}

func benchInsertManyNaive(b *testing.B, n int) {
	values := generateSortedUniqueBigInts(n, int64(n))
	rand.New(rand.NewSource(int64(n))).Shuffle(len(values), func(i, j int) {
		values[i], values[j] = values[j], values[i]
	})
	h := NewPoseidonHasher()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree := New(h)
		if err := tree.InsertMany(values); err != nil {
			b.Fatal(err)
		}
	}
}
