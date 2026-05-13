package snapshot

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/leanimt_plus"
)

func TestBinaryRoundTrip(t *testing.T) {
	h := leanimt_plus.NewPoseidonHasher()
	tree := buildLeanTree(t)
	originalRoot := tree.Root()
	originalDepth := tree.Depth()
	originalLeafCount := tree.LeafCount()
	var crlNum uint64 = 42

	var buf bytes.Buffer
	if err := ExportBinary(tree, crlNum, &buf); err != nil {
		t.Fatal(err)
	}
	restored, gotCRL, err := ImportBinary(h, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Root().Cmp(originalRoot) != 0 {
		t.Errorf("root mismatch: got %s want %s", restored.Root().Text(16), originalRoot.Text(16))
	}
	if restored.Depth() != originalDepth {
		t.Errorf("depth: got %d want %d", restored.Depth(), originalDepth)
	}
	if restored.LeafCount() != originalLeafCount {
		t.Errorf("leafCount: got %d want %d", restored.LeafCount(), originalLeafCount)
	}
	if gotCRL != crlNum {
		t.Errorf("crl: got %d want %d", gotCRL, crlNum)
	}

	member, _ := new(big.Int).SetString("100048210DD2DF2E128096A9282B5EC5", 16)
	proof, err := restored.GenerateProof(member)
	if err != nil {
		t.Fatal(err)
	}
	if proof.ProofType != leanimt_plus.ProofMembership {
		t.Fatal("expected membership")
	}
	if !leanimt_plus.VerifyProof(h, proof) {
		t.Fatal("verify failed")
	}
}

func TestBinaryEmptyTree(t *testing.T) {
	h := leanimt_plus.NewPoseidonHasher()
	tree := leanimt_plus.New(h)

	var buf bytes.Buffer
	if err := ExportBinary(tree, 0, &buf); err != nil {
		t.Fatal(err)
	}
	restored, _, err := ImportBinary(h, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Root() != nil {
		t.Errorf("empty root: got %v", restored.Root())
	}
	if restored.Size() != 0 {
		t.Errorf("empty size: got %d", restored.Size())
	}
}

func TestBinaryCrossFormat(t *testing.T) {
	h := leanimt_plus.NewPoseidonHasher()
	tree := buildLeanTree(t)

	var jsonBuf bytes.Buffer
	if err := Export(tree, 99, &jsonBuf); err != nil {
		t.Fatal(err)
	}
	jsonTree, _, err := Import(h, &jsonBuf)
	if err != nil {
		t.Fatal(err)
	}

	var binBuf bytes.Buffer
	if err := ExportBinary(jsonTree, 99, &binBuf); err != nil {
		t.Fatal(err)
	}
	binTree, _, err := ImportBinary(h, &binBuf)
	if err != nil {
		t.Fatal(err)
	}

	if jsonTree.Root().Cmp(binTree.Root()) != 0 {
		t.Errorf("root mismatch: json=%s bin=%s", jsonTree.Root().Text(16), binTree.Root().Text(16))
	}

	member, _ := new(big.Int).SetString("200048210DD2DF2E128096A9282B5EC5", 16)
	jp, _ := jsonTree.GenerateProof(member)
	bp, _ := binTree.GenerateProof(member)
	if jp.ProofType != bp.ProofType || len(jp.Siblings) != len(bp.Siblings) || jp.LeafIndex != bp.LeafIndex {
		t.Errorf("proof mismatch between formats")
	}
}

func TestBinaryTruncated(t *testing.T) {
	h := leanimt_plus.NewPoseidonHasher()
	tree := buildLeanTree(t)
	var buf bytes.Buffer
	if err := ExportBinary(tree, 0, &buf); err != nil {
		t.Fatal(err)
	}
	truncated := buf.Bytes()[:BinaryHeader+1]
	if _, _, err := ImportBinary(h, bytes.NewReader(truncated)); err == nil {
		t.Fatal("expected truncation error")
	}
}

func TestBinaryInvalidMagic(t *testing.T) {
	h := leanimt_plus.NewPoseidonHasher()
	var buf [BinaryHeader]byte
	binary.BigEndian.PutUint16(buf[0:2], 0xDEAD)
	binary.BigEndian.PutUint16(buf[2:4], BinaryVersion)
	if _, _, err := ImportBinary(h, bytes.NewReader(buf[:])); err == nil {
		t.Fatal("expected magic error")
	}
}

func TestBinaryUnknownVersion(t *testing.T) {
	h := leanimt_plus.NewPoseidonHasher()
	var buf [BinaryHeader]byte
	binary.BigEndian.PutUint16(buf[0:2], BinaryMagic)
	binary.BigEndian.PutUint16(buf[2:4], 99)
	if _, _, err := ImportBinary(h, bytes.NewReader(buf[:])); err == nil {
		t.Fatal("expected version error")
	}
}

func TestBinaryFileRoundTrip(t *testing.T) {
	h := leanimt_plus.NewPoseidonHasher()
	tree := buildLeanTree(t)
	dir := t.TempDir()

	rawPath := filepath.Join(dir, "test.bin")
	if err := ExportBinaryFile(tree, 42, rawPath); err != nil {
		t.Fatal(err)
	}
	restored, crl, err := ImportBinaryFile(h, rawPath)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Root().Cmp(tree.Root()) != 0 {
		t.Error("root mismatch (uncompressed)")
	}
	if crl != 42 {
		t.Errorf("crl: got %d", crl)
	}

	gzPath := filepath.Join(dir, "test.bin.gz")
	if err := ExportBinaryFile(tree, 42, gzPath); err != nil {
		t.Fatal(err)
	}
	restored2, _, err := ImportBinaryFile(h, gzPath)
	if err != nil {
		t.Fatal(err)
	}
	if restored2.Root().Cmp(tree.Root()) != 0 {
		t.Error("root mismatch (compressed)")
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("stale temp file: %s", e.Name())
		}
	}
}
