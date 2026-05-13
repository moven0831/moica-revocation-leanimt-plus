package snapshot

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"

	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/hexenc"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/leanimt"
)

const SnapshotVersion = 2

// Snapshot is the v2 LeanIMT+ JSON-on-the-wire form. The Nodes grid is
// [level][index], and Leaves[0] is the sentinel.
type Snapshot struct {
	Version   int         `json:"version"`
	Root      string      `json:"root"`
	Depth     int         `json:"depth"`
	LeafCount int         `json:"leafCount"`
	Size      int         `json:"size"`
	CRLNumber uint64      `json:"crlNumber"`
	Nodes     [][]string  `json:"nodes"`
	Leaves    []LeafEntry `json:"leaves"`
}

type LeafEntry struct {
	Value     string `json:"value"`
	NextValue string `json:"nextValue"`
}

func Export(tree *leanimt.LeanIMTPlus, crlNumber uint64, w io.Writer) error {
	nodes, leaves := tree.ExportState()

	jsonNodes := make([][]string, len(nodes))
	for lvl, level := range nodes {
		jsonNodes[lvl] = make([]string, len(level))
		for i, n := range level {
			if n == nil {
				jsonNodes[lvl][i] = ""
			} else {
				jsonNodes[lvl][i] = hexenc.Encode(n)
			}
		}
	}
	jsonLeaves := make([]LeafEntry, len(leaves))
	for i, l := range leaves {
		jsonLeaves[i] = LeafEntry{Value: hexenc.Encode(l.Value), NextValue: hexenc.Encode(l.NextValue)}
	}

	rootStr := hexenc.Encode(tree.Root())

	snap := Snapshot{
		Version:   SnapshotVersion,
		Root:      rootStr,
		Depth:     tree.Depth(),
		LeafCount: tree.LeafCount(),
		Size:      tree.Size(),
		CRLNumber: crlNumber,
		Nodes:     jsonNodes,
		Leaves:    jsonLeaves,
	}

	gw := gzip.NewWriter(w)
	defer gw.Close()
	return json.NewEncoder(gw).Encode(snap)
}

func ExportFile(tree *leanimt.LeanIMTPlus, crlNumber uint64, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), "snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	tmpPath := f.Name()

	if err := Export(tree, crlNumber, f); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("export: %w", err)
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

func ImportFile(h leanimt.Hasher, path string) (*leanimt.LeanIMTPlus, uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	return Import(h, f)
}

func Import(h leanimt.Hasher, r io.Reader) (*leanimt.LeanIMTPlus, uint64, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, 0, fmt.Errorf("gzip reader: %w", err)
	}
	defer gr.Close()

	var snap Snapshot
	if err := json.NewDecoder(gr).Decode(&snap); err != nil {
		return nil, 0, fmt.Errorf("json decode: %w", err)
	}
	if snap.Version != SnapshotVersion {
		return nil, 0, fmt.Errorf("unsupported snapshot version: %d (expected %d)", snap.Version, SnapshotVersion)
	}
	if snap.Depth < 0 || (len(snap.Nodes) > 0 && snap.Depth != len(snap.Nodes)-1) {
		return nil, 0, fmt.Errorf("snapshot depth %d disagrees with %d node levels", snap.Depth, len(snap.Nodes))
	}

	nodes := make([][]*big.Int, len(snap.Nodes))
	for lvl, level := range snap.Nodes {
		nodes[lvl] = make([]*big.Int, len(level))
		for i, s := range level {
			if s == "" {
				continue
			}
			n, err := hexenc.Decode(s)
			if err != nil {
				return nil, 0, fmt.Errorf("nodes[%d][%d]: %w", lvl, i, err)
			}
			nodes[lvl][i] = n
		}
	}
	leaves := make([]leanimt.IndexedLeaf, len(snap.Leaves))
	for i, l := range snap.Leaves {
		v, err := hexenc.Decode(l.Value)
		if err != nil {
			return nil, 0, fmt.Errorf("leaves[%d].value: %w", i, err)
		}
		nv, err := hexenc.Decode(l.NextValue)
		if err != nil {
			return nil, 0, fmt.Errorf("leaves[%d].nextValue: %w", i, err)
		}
		leaves[i] = leanimt.IndexedLeaf{Value: v, NextValue: nv}
	}

	tree := leanimt.New(h)
	if len(leaves) > 0 {
		if err := tree.ImportState(nodes, leaves); err != nil {
			return nil, 0, fmt.Errorf("import state: %w", err)
		}
	}
	return tree, snap.CRLNumber, nil
}
