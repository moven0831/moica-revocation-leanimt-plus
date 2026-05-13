package snapshot

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/moven0831/moica-revocation-smt/server/internal/leanimt"
)

func bigsFromHex(t *testing.T, hexes ...string) []*big.Int {
	t.Helper()
	out := make([]*big.Int, len(hexes))
	for i, h := range hexes {
		n, ok := new(big.Int).SetString(h, 16)
		if !ok {
			t.Fatalf("invalid hex: %s", h)
		}
		out[i] = n
	}
	return out
}

func buildLeanTree(t *testing.T) *leanimt.LeanIMTPlus {
	t.Helper()
	h := leanimt.NewPoseidonHasher()
	tree := leanimt.New(h)
	serials := bigsFromHex(t,
		"100048210DD2DF2E128096A9282B5EC5",
		"200048210DD2DF2E128096A9282B5EC5",
		"300048210DD2DF2E128096A9282B5EC5",
	)
	if err := tree.InsertManySorted(serials); err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestSnapshotRoundTrip(t *testing.T) {
	h := leanimt.NewPoseidonHasher()
	tree := buildLeanTree(t)

	originalRoot := tree.Root()
	originalSize := tree.Size()
	originalDepth := tree.Depth()

	var buf bytes.Buffer
	if err := Export(tree, 0, &buf); err != nil {
		t.Fatal("export:", err)
	}

	restored, _, err := Import(h, &buf)
	if err != nil {
		t.Fatal("import:", err)
	}
	if restored.Root().Cmp(originalRoot) != 0 {
		t.Errorf("root mismatch: got %s, want %s", restored.Root().Text(16), originalRoot.Text(16))
	}
	if restored.Size() != originalSize {
		t.Errorf("size mismatch: got %d, want %d", restored.Size(), originalSize)
	}
	if restored.Depth() != originalDepth {
		t.Errorf("depth mismatch: got %d, want %d", restored.Depth(), originalDepth)
	}

	member := bigsFromHex(t, "100048210DD2DF2E128096A9282B5EC5")[0]
	proof, err := restored.GenerateProof(member)
	if err != nil {
		t.Fatal(err)
	}
	if proof.ProofType != leanimt.ProofMembership {
		t.Fatal("expected membership proof")
	}
	if !leanimt.VerifyProof(h, proof) {
		t.Fatal("proof verification failed on restored tree")
	}

	nonMember := bigsFromHex(t, "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")[0]
	nonProof, err := restored.GenerateProof(nonMember)
	if err != nil {
		t.Fatal(err)
	}
	if nonProof.ProofType != leanimt.ProofNonMembership {
		t.Fatal("expected non-membership proof")
	}
	if !leanimt.VerifyProof(h, nonProof) {
		t.Fatal("non-membership verification failed")
	}
}

func TestExportFileRoundTrip(t *testing.T) {
	h := leanimt.NewPoseidonHasher()
	tree := buildLeanTree(t)
	originalRoot := tree.Root()
	var crlNum uint64 = 99

	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "tree-snapshot.json.gz")
	if err := ExportFile(tree, crlNum, path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}

	restored, gotCRL, err := ImportFile(h, path)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Root().Cmp(originalRoot) != 0 {
		t.Errorf("root mismatch")
	}
	if gotCRL != crlNum {
		t.Errorf("crlNumber: got %d, want %d", gotCRL, crlNum)
	}

	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("stale temp file found: %s", e.Name())
		}
	}
}

func TestSnapshotEmpty(t *testing.T) {
	h := leanimt.NewPoseidonHasher()
	tree := leanimt.New(h)

	var buf bytes.Buffer
	if err := Export(tree, 0, &buf); err != nil {
		t.Fatal(err)
	}
	restored, _, err := Import(h, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Root() != nil {
		t.Errorf("empty tree root should be nil, got %v", restored.Root())
	}
	if restored.Size() != 0 {
		t.Errorf("empty Size: got %d", restored.Size())
	}
}

func TestSnapshotCRLNumber(t *testing.T) {
	h := leanimt.NewPoseidonHasher()
	tree := leanimt.New(h)
	if err := tree.Insert(big.NewInt(0xABCDEF)); err != nil {
		t.Fatal(err)
	}
	var crlNum uint64 = 42
	var buf bytes.Buffer
	if err := Export(tree, crlNum, &buf); err != nil {
		t.Fatal(err)
	}
	restored, gotCRL, err := Import(h, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if gotCRL != crlNum {
		t.Errorf("got %d want %d", gotCRL, crlNum)
	}
	if restored.Root().Cmp(tree.Root()) != 0 {
		t.Errorf("root mismatch")
	}
}

func TestSnapshotRejectsWrongVersion(t *testing.T) {
	h := leanimt.NewPoseidonHasher()
	v1 := map[string]any{"version": 1, "root": "0x0", "nodes": []any{}}
	body, _ := json.Marshal(v1)
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write(body)
	gw.Close()
	if _, _, err := Import(h, &buf); err == nil {
		t.Fatal("expected version mismatch error")
	}
}
