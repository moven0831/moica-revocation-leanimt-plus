package crl

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/leanimt"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/manager"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/snapshot"
)

type IssuerConfig struct {
	ID  string
	URL string
}

type Watcher struct {
	interval time.Duration
	issuers  []IssuerConfig
	mgr      *manager.TreeManager
	hasher   leanimt.Hasher
	dataDir  string
	wg       sync.WaitGroup
}

func NewWatcher(interval time.Duration, issuers []IssuerConfig, mgr *manager.TreeManager, hasher leanimt.Hasher, dataDir string) *Watcher {
	return &Watcher{
		interval: interval,
		issuers:  issuers,
		mgr:      mgr,
		hasher:   hasher,
		dataDir:  dataDir,
	}
}

func (w *Watcher) Wait() { w.wg.Wait() }

func (w *Watcher) Start(ctx context.Context) {
	w.fetchAll()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.fetchAll()
		}
	}
}

func (w *Watcher) fetchAll() {
	for _, issuer := range w.issuers {
		if err := w.fetchAndRebuild(issuer); err != nil {
			log.Printf("CRL fetch error for %s: %v", issuer.ID, err)
		}
	}
}

func (w *Watcher) fetchAndRebuild(issuer IssuerConfig) error {
	log.Printf("Fetching CRL for %s from %s", issuer.ID, issuer.URL)

	derBytes, err := FetchDER(issuer.URL)
	if err != nil {
		return err
	}

	parsed, err := ParseDER(derBytes)
	if err != nil {
		return err
	}

	existing, _ := w.mgr.GetTree(issuer.ID)
	if existing != nil && parsed.CRLNumber != nil {
		if parsed.CRLNumber.Uint64() <= existing.CRLNumber {
			log.Printf("CRL for %s is not newer (have %d, got %d), skipping",
				issuer.ID, existing.CRLNumber, parsed.CRLNumber.Uint64())
			return nil
		}
	}

	serials := DedupAndSortSerials(parsed.RevokedSerials)
	log.Printf("Building LeanIMT+ for %s with %d unique sorted serials", issuer.ID, len(serials))

	tree := leanimt.New(w.hasher)
	if len(serials) > 0 {
		if err := tree.InsertManySorted(serials); err != nil {
			return err
		}
	}

	var crlNum uint64
	if parsed.CRLNumber != nil {
		crlNum = parsed.CRLNumber.Uint64()
	}

	w.mgr.SetTree(issuer.ID, tree, crlNum)
	rootStr := "nil"
	if r := tree.Root(); r != nil {
		rootStr = r.Text(16)
		if len(rootStr) > 16 {
			rootStr = rootStr[:16] + "..."
		}
	}
	log.Printf("LeanIMT+ for %s loaded: root=%s size=%d depth=%d crl=%d",
		issuer.ID, rootStr, tree.Size(), tree.Depth(), crlNum)

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		snapPath := filepath.Join(w.dataDir, issuer.ID, "tree-snapshot.json.gz")
		if err := snapshot.ExportFile(tree, crlNum, snapPath); err != nil {
			log.Printf("Snapshot export failed for %s: %v", issuer.ID, err)
		} else {
			log.Printf("Snapshot exported for %s to %s", issuer.ID, snapPath)
		}
	}()

	return nil
}
