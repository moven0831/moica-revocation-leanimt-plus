package snapshot

import (
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/leanimt"
)

const (
	BinaryMagic   = 0x4C49 // "LI"
	BinaryVersion = 2
	BinaryHeader  = 50 // 2+2+2+4+8+32
)

// ExportBinary serializes a LeanIMT+ tree. The encoded form is:
//
//	HEADER (50 bytes):
//	  magic       uint16  0x4C49 ("LI")
//	  version     uint16  2
//	  depth       uint16
//	  leafCount   uint32
//	  crlNumber   uint64
//	  root        [32]byte
//	LEVELS (depth+1 of them):
//	  count       uint32
//	  per entry:  present uint8 (0|1); hash [32]byte (zero if absent)
//	LEAVES (leafCount of them, includes sentinel):
//	  value       [32]byte
//	  nextValue   [32]byte
func ExportBinary(tree *leanimt.LeanIMTPlus, crlNumber uint64, w io.Writer) error {
	nodes, leaves := tree.ExportState()

	var hdr [BinaryHeader]byte
	binary.BigEndian.PutUint16(hdr[0:2], BinaryMagic)
	binary.BigEndian.PutUint16(hdr[2:4], BinaryVersion)
	binary.BigEndian.PutUint16(hdr[4:6], uint16(tree.Depth()))
	binary.BigEndian.PutUint32(hdr[6:10], uint32(tree.LeafCount()))
	binary.BigEndian.PutUint64(hdr[10:18], crlNumber)
	rootBytes := bigTo32(tree.Root())
	copy(hdr[18:50], rootBytes[:])
	if _, err := w.Write(hdr[:]); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for lvl, level := range nodes {
		var lenBuf [4]byte
		binary.BigEndian.PutUint32(lenBuf[:], uint32(len(level)))
		if _, err := w.Write(lenBuf[:]); err != nil {
			return fmt.Errorf("write level %d count: %w", lvl, err)
		}
		for i, n := range level {
			present := byte(0)
			if n != nil {
				present = 1
			}
			if _, err := w.Write([]byte{present}); err != nil {
				return fmt.Errorf("write level %d entry %d presence: %w", lvl, i, err)
			}
			b := bigTo32(n)
			if _, err := w.Write(b[:]); err != nil {
				return fmt.Errorf("write level %d entry %d hash: %w", lvl, i, err)
			}
		}
	}

	for i, l := range leaves {
		v := bigTo32(l.Value)
		nv := bigTo32(l.NextValue)
		if _, err := w.Write(v[:]); err != nil {
			return fmt.Errorf("write leaf %d value: %w", i, err)
		}
		if _, err := w.Write(nv[:]); err != nil {
			return fmt.Errorf("write leaf %d nextValue: %w", i, err)
		}
	}

	return nil
}

func ImportBinary(h leanimt.Hasher, r io.Reader) (*leanimt.LeanIMTPlus, uint64, error) {
	var hdr [BinaryHeader]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, 0, fmt.Errorf("read header: %w", err)
	}
	magic := binary.BigEndian.Uint16(hdr[0:2])
	if magic != BinaryMagic {
		return nil, 0, fmt.Errorf("invalid magic: 0x%04X (expected 0x%04X)", magic, BinaryMagic)
	}
	version := binary.BigEndian.Uint16(hdr[2:4])
	if version != BinaryVersion {
		return nil, 0, fmt.Errorf("unsupported version: %d (expected %d)", version, BinaryVersion)
	}
	depth := int(binary.BigEndian.Uint16(hdr[4:6]))
	leafCount := int(binary.BigEndian.Uint32(hdr[6:10]))
	crlNumber := binary.BigEndian.Uint64(hdr[10:18])

	nodes := make([][]*big.Int, depth+1)
	var lenBuf [4]byte
	var presBuf [1]byte
	var hashBuf [32]byte
	for lvl := 0; lvl <= depth; lvl++ {
		if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
			return nil, 0, fmt.Errorf("read level %d count: %w", lvl, err)
		}
		count := int(binary.BigEndian.Uint32(lenBuf[:]))
		level := make([]*big.Int, count)
		for i := 0; i < count; i++ {
			if _, err := io.ReadFull(r, presBuf[:]); err != nil {
				return nil, 0, fmt.Errorf("read level %d entry %d presence: %w", lvl, i, err)
			}
			if _, err := io.ReadFull(r, hashBuf[:]); err != nil {
				return nil, 0, fmt.Errorf("read level %d entry %d hash: %w", lvl, i, err)
			}
			if presBuf[0] == 1 {
				level[i] = new(big.Int).SetBytes(hashBuf[:])
			}
		}
		nodes[lvl] = level
	}

	leaves := make([]leanimt.IndexedLeaf, leafCount)
	var valBuf, nextBuf [32]byte
	for i := 0; i < leafCount; i++ {
		if _, err := io.ReadFull(r, valBuf[:]); err != nil {
			return nil, 0, fmt.Errorf("read leaf %d value: %w", i, err)
		}
		if _, err := io.ReadFull(r, nextBuf[:]); err != nil {
			return nil, 0, fmt.Errorf("read leaf %d nextValue: %w", i, err)
		}
		leaves[i] = leanimt.IndexedLeaf{
			Value:     new(big.Int).SetBytes(valBuf[:]),
			NextValue: new(big.Int).SetBytes(nextBuf[:]),
		}
	}

	tree := leanimt.New(h)
	if leafCount > 0 {
		if err := tree.ImportState(nodes, leaves); err != nil {
			return nil, 0, fmt.Errorf("import state: %w", err)
		}
	}
	return tree, crlNumber, nil
}

func ExportBinaryFile(tree *leanimt.LeanIMTPlus, crlNumber uint64, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), "snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	tmpPath := f.Name()

	var w io.Writer = f
	var gw *gzip.Writer
	if strings.HasSuffix(path, ".gz") {
		gw = gzip.NewWriter(f)
		w = gw
	}

	if err := ExportBinary(tree, crlNumber, w); err != nil {
		if gw != nil {
			gw.Close()
		}
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("export: %w", err)
	}
	if gw != nil {
		if err := gw.Close(); err != nil {
			f.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("gzip close: %w", err)
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

func ImportBinaryFile(h leanimt.Hasher, path string) (*leanimt.LeanIMTPlus, uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return nil, 0, fmt.Errorf("gzip reader: %w", err)
		}
		defer gr.Close()
		r = gr
	}
	return ImportBinary(h, r)
}

func bigTo32(n *big.Int) [32]byte {
	var buf [32]byte
	if n != nil && n.Sign() > 0 {
		n.FillBytes(buf[:])
	}
	return buf
}
