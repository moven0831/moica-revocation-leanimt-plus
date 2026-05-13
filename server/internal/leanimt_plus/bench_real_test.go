//go:build bench_real

package leanimt_plus_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/crl"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/leanimt_plus"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/snapshot"
)

// Must match bench/smt-bench-pr/bench_real_test.go so both impls measure on
// identical inputs.
const (
	benchQueryCount = 1024
	benchSeed       = 42
)

type datasetID string

const (
	datasetG2 datasetID = "G2"
	datasetG3 datasetID = "G3"
)

type loadedDataset struct {
	once    sync.Once
	serials []*big.Int
	err     error
}

var datasets = map[datasetID]*loadedDataset{
	datasetG2: {},
	datasetG3: {},
}

func derFilename(id datasetID) string {
	switch id {
	case datasetG2:
		return "g2.crl.der"
	case datasetG3:
		return "g3.crl.der"
	default:
		return ""
	}
}

// Skips loudly if MOICA_BENCH_DER_DIR is unset — never falls back to live
// HTTP, because both repos must consume the same bytes.
func loadSerials(tb testing.TB, id datasetID) []*big.Int {
	dir := os.Getenv("MOICA_BENCH_DER_DIR")
	if dir == "" {
		tb.Skipf("MOICA_BENCH_DER_DIR not set; run via bench/run.sh which writes DERs to bench/.cache/")
	}
	entry := datasets[id]
	entry.once.Do(func() {
		path := filepath.Join(dir, derFilename(id))
		raw, err := os.ReadFile(path)
		if err != nil {
			entry.err = fmt.Errorf("read %s: %w", path, err)
			return
		}
		parsed, err := crl.ParseDER(raw)
		if err != nil {
			entry.err = fmt.Errorf("parse %s: %w", path, err)
			return
		}
		entry.serials = crl.DedupAndSortSerials(parsed.RevokedSerials)
	})
	if entry.err != nil {
		tb.Fatal(entry.err)
	}
	if len(entry.serials) == 0 {
		tb.Fatalf("dataset %s loaded empty", id)
	}
	return entry.serials
}

func membershipQueries(serials []*big.Int, k int, seed int64) []*big.Int {
	r := rand.New(rand.NewSource(seed))
	out := make([]*big.Int, k)
	for i := range out {
		out[i] = new(big.Int).Set(serials[r.Intn(len(serials))])
	}
	return out
}

func nonMembershipQueries(serials []*big.Int, k int, seed int64) []*big.Int {
	r := rand.New(rand.NewSource(seed))
	maxBits := serials[len(serials)-1].BitLen()
	if maxBits < 8 {
		maxBits = 8
	}
	bound := new(big.Int).Lsh(big.NewInt(1), uint(maxBits))
	out := make([]*big.Int, 0, k)
	for len(out) < k {
		v := new(big.Int).Rand(r, bound)
		if v.Sign() == 0 {
			continue
		}
		idx := sort.Search(len(serials), func(i int) bool { return serials[i].Cmp(v) >= 0 })
		if idx < len(serials) && serials[idx].Cmp(v) == 0 {
			continue
		}
		out = append(out, v)
	}
	return out
}

func newTreeFor(tb testing.TB, serials []*big.Int) *leanimt_plus.LeanIMTPlus {
	h := leanimt_plus.NewPoseidonHasher()
	tree := leanimt_plus.New(h)
	if err := tree.InsertManySorted(serials); err != nil {
		tb.Fatalf("build tree: %v", err)
	}
	return tree
}

func BenchmarkReal_Build(b *testing.B) {
	for _, ds := range []datasetID{datasetG2, datasetG3} {
		ds := ds
		b.Run(string(ds), func(b *testing.B) {
			serials := loadSerials(b, ds)
			h := leanimt_plus.NewPoseidonHasher()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tree := leanimt_plus.New(h)
				if err := tree.InsertManySorted(serials); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkReal_ProofGen(b *testing.B) {
	for _, ds := range []datasetID{datasetG2, datasetG3} {
		ds := ds
		b.Run(string(ds), func(b *testing.B) {
			serials := loadSerials(b, ds)
			tree := newTreeFor(b, serials)
			memQ := membershipQueries(serials, benchQueryCount, benchSeed)
			nonQ := nonMembershipQueries(serials, benchQueryCount, benchSeed)

			b.Run("Membership", func(b *testing.B) { benchProofGen(b, tree, memQ) })
			b.Run("NonMembership", func(b *testing.B) { benchProofGen(b, tree, nonQ) })
		})
	}
}

func benchProofGen(b *testing.B, tree *leanimt_plus.LeanIMTPlus, queries []*big.Int) {
	// Generate one proof outside the timer so its shape can be reported as a metric.
	sample, err := tree.GenerateProof(queries[0])
	if err != nil {
		b.Fatalf("sample proof: %v", err)
	}
	blob, err := json.Marshal(sample)
	if err != nil {
		b.Fatalf("marshal sample: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := tree.GenerateProof(queries[i%len(queries)]); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(len(sample.Siblings)), "siblings/op")
	b.ReportMetric(float64(len(blob)), "proofBytes/op")
}

func BenchmarkReal_Verify(b *testing.B) {
	for _, ds := range []datasetID{datasetG2, datasetG3} {
		ds := ds
		b.Run(string(ds), func(b *testing.B) {
			serials := loadSerials(b, ds)
			h := leanimt_plus.NewPoseidonHasher()
			tree := newTreeFor(b, serials)
			memQ := membershipQueries(serials, benchQueryCount, benchSeed)
			nonQ := nonMembershipQueries(serials, benchQueryCount, benchSeed)
			memProofs := preGenerate(b, tree, memQ)
			nonProofs := preGenerate(b, tree, nonQ)

			b.Run("Membership", func(b *testing.B) { benchVerify(b, h, memProofs) })
			b.Run("NonMembership", func(b *testing.B) { benchVerify(b, h, nonProofs) })
		})
	}
}

func preGenerate(b *testing.B, tree *leanimt_plus.LeanIMTPlus, queries []*big.Int) []*leanimt_plus.Proof {
	out := make([]*leanimt_plus.Proof, len(queries))
	for i, q := range queries {
		p, err := tree.GenerateProof(q)
		if err != nil {
			b.Fatalf("pre-generate proof: %v", err)
		}
		out[i] = p
	}
	return out
}

func benchVerify(b *testing.B, h leanimt_plus.Hasher, proofs []*leanimt_plus.Proof) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !leanimt_plus.VerifyProof(h, proofs[i%len(proofs)]) {
			b.Fatal("proof failed to verify")
		}
	}
}

// Kept off the main timing benches: atomic.Add adds ~5-10 ns/op.
type countingHasher struct {
	inner leanimt_plus.Hasher
	n2    atomic.Uint64
}

func (c *countingHasher) Hash2(a, b *big.Int) *big.Int {
	c.n2.Add(1)
	return c.inner.Hash2(a, b)
}

func BenchmarkReal_HashCount_Build(b *testing.B) {
	for _, ds := range []datasetID{datasetG2, datasetG3} {
		ds := ds
		b.Run(string(ds), func(b *testing.B) {
			serials := loadSerials(b, ds)
			c := &countingHasher{inner: leanimt_plus.NewPoseidonHasher()}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				c.n2.Store(0)
				tree := leanimt_plus.New(c)
				if err := tree.InsertManySorted(serials); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(c.n2.Load()), "hash2/op")
		})
	}
}

func BenchmarkReal_HashCount_Verify(b *testing.B) {
	for _, ds := range []datasetID{datasetG2, datasetG3} {
		ds := ds
		b.Run(string(ds), func(b *testing.B) {
			serials := loadSerials(b, ds)
			h := leanimt_plus.NewPoseidonHasher()
			tree := newTreeFor(b, serials)
			memQ := membershipQueries(serials, benchQueryCount, benchSeed)
			nonQ := nonMembershipQueries(serials, benchQueryCount, benchSeed)
			memProofs := preGenerate(b, tree, memQ)
			nonProofs := preGenerate(b, tree, nonQ)
			c := &countingHasher{inner: h}

			b.Run("Membership", func(b *testing.B) {
				c.n2.Store(0)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if !leanimt_plus.VerifyProof(c, memProofs[i%len(memProofs)]) {
						b.Fatal("verify failed")
					}
				}
				b.StopTimer()
				if b.N > 0 {
					b.ReportMetric(float64(c.n2.Load())/float64(b.N), "hash2/op")
				}
			})
			b.Run("NonMembership", func(b *testing.B) {
				c.n2.Store(0)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if !leanimt_plus.VerifyProof(c, nonProofs[i%len(nonProofs)]) {
						b.Fatal("verify failed")
					}
				}
				b.StopTimer()
				if b.N > 0 {
					b.ReportMetric(float64(c.n2.Load())/float64(b.N), "hash2/op")
				}
			})
		})
	}
}

func TestReal_SnapshotSize(t *testing.T) {
	tmp := t.TempDir()
	for _, ds := range []datasetID{datasetG2, datasetG3} {
		serials := loadSerials(t, ds)
		tree := newTreeFor(t, serials)

		gzPath := filepath.Join(tmp, fmt.Sprintf("%s.json.gz", ds))
		if err := snapshot.ExportFile(tree, 0, gzPath); err != nil {
			t.Fatalf("%s: export: %v", ds, err)
		}
		gzBytes, err := os.ReadFile(gzPath)
		if err != nil {
			t.Fatalf("%s: read gz: %v", ds, err)
		}
		gr, err := gzip.NewReader(bytes.NewReader(gzBytes))
		if err != nil {
			t.Fatalf("%s: gunzip: %v", ds, err)
		}
		rawBytes, err := io.ReadAll(gr)
		gr.Close()
		if err != nil {
			t.Fatalf("%s: read raw: %v", ds, err)
		}

		// Format parsed by bench/cmd/report.
		t.Logf("SNAPSHOT %s gz=%d raw=%d leaves=%d depth=%d format=json.gz",
			ds, len(gzBytes), len(rawBytes), tree.LeafCount(), tree.Depth())
	}
}
