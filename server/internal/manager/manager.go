package manager

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/leanimt"
)

type TreeEntry struct {
	Tree      *leanimt.LeanIMTPlus
	CRLNumber uint64
	LoadedAt  time.Time
}

type TreeManager struct {
	mu     sync.RWMutex
	trees  map[string]*TreeEntry
	hasher leanimt.Hasher
}

func New(h leanimt.Hasher) *TreeManager {
	return &TreeManager{
		trees:  make(map[string]*TreeEntry),
		hasher: h,
	}
}

func (m *TreeManager) Hasher() leanimt.Hasher {
	return m.hasher
}

func (m *TreeManager) GetTree(issuerID string) (*TreeEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.trees[issuerID]
	if !ok {
		return nil, fmt.Errorf("unknown issuer: %s", issuerID)
	}
	return entry, nil
}

func (m *TreeManager) SetTree(issuerID string, tree *leanimt.LeanIMTPlus, crlNumber uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trees[issuerID] = &TreeEntry{
		Tree:      tree,
		CRLNumber: crlNumber,
		LoadedAt:  time.Now(),
	}
}

func (m *TreeManager) GetProof(issuerID string, serialNumber *big.Int) (*leanimt.Proof, error) {
	m.mu.RLock()
	entry, ok := m.trees[issuerID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown issuer: %s", issuerID)
	}
	return entry.Tree.GenerateProof(serialNumber)
}

func (m *TreeManager) IssuerIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.trees))
	for id := range m.trees {
		ids = append(ids, id)
	}
	return ids
}

type IssuerStatus struct {
	Loaded    bool   `json:"loaded"`
	Size      int    `json:"size"`
	LeafCount int    `json:"leafCount"`
	Depth     int    `json:"depth"`
	Root      string `json:"root"`
	CRLNumber uint64 `json:"crlNumber"`
	LoadedAt  string `json:"loadedAt"`
}

func (m *TreeManager) Status() map[string]IssuerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]IssuerStatus)
	for id, entry := range m.trees {
		root := "0x0"
		if r := entry.Tree.Root(); r != nil && r.Sign() != 0 {
			root = "0x" + r.Text(16)
		}
		status[id] = IssuerStatus{
			Loaded:    true,
			Size:      entry.Tree.Size(),
			LeafCount: entry.Tree.LeafCount(),
			Depth:     entry.Tree.Depth(),
			Root:      root,
			CRLNumber: entry.CRLNumber,
			LoadedAt:  entry.LoadedAt.UTC().Format(time.RFC3339),
		}
	}
	return status
}
