package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/chain"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/config"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/crl"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/leanimt"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/snapshot"
)

type issuer struct {
	ID  string
	URL string
}

type rootInfo struct {
	Root      string `json:"root"`
	Size      int    `json:"size"`
	LeafCount int    `json:"leafCount"`
	Depth     int    `json:"depth"`
	CRLNumber string `json:"crlNumber"`
	Timestamp string `json:"timestamp"`
}

func main() {
	postRoot := flag.Bool("post-root", false, "Post LeanIMT+ roots on-chain (reads root.json files, skips tree build)")
	exportBinary := flag.Bool("binary", false, "Also export binary format snapshot alongside JSON")
	convertBinary := flag.String("convert-binary", "", "Convert JSON snapshot to binary format (path to .json.gz input)")
	flag.Parse()

	cfg := config.Load()

	if *convertBinary != "" {
		convertJSONToBinary(*convertBinary)
		return
	}

	if *postRoot {
		postRootOnChain(cfg)
		return
	}

	issuers := []issuer{
		{ID: "g2", URL: cfg.CRLG2URL},
		{ID: "g3", URL: cfg.CRLG3URL},
	}

	hasher := leanimt.NewPoseidonHasher()
	anyChanged := false

	for _, iss := range issuers {
		log.Printf("[%s] Fetching CRL from %s", iss.ID, iss.URL)
		derBytes, err := crl.FetchDER(iss.URL)
		if err != nil {
			log.Printf("[%s] Skipping: %v", iss.ID, err)
			continue
		}
		log.Printf("[%s] Fetched %d bytes", iss.ID, len(derBytes))

		parsed, err := crl.ParseDER(derBytes)
		if err != nil {
			log.Printf("[%s] Skipping: parse error: %v", iss.ID, err)
			continue
		}
		log.Printf("[%s] Parsed %d revoked serials (CRLNumber=%s)",
			iss.ID, len(parsed.RevokedSerials), parsed.CRLNumber)

		serials := crl.DedupAndSortSerials(parsed.RevokedSerials)
		log.Printf("[%s] %d unique sorted serials", iss.ID, len(serials))

		buildStart := time.Now()
		tree := leanimt.New(hasher)
		if len(serials) > 0 {
			err = tree.InsertManyWithProgress(serials, 10_000, func(done, total int) {
				log.Printf("[%s] Inserted %d / %d", iss.ID, done, total)
			})
			if err != nil {
				log.Printf("[%s] Skipping: build error: %v", iss.ID, err)
				continue
			}
		}
		rootHex := "0x0"
		if r := tree.Root(); r != nil {
			rootHex = "0x" + r.Text(16)
		}
		log.Printf("[%s] Tree ready: size=%d depth=%d root=%s duration=%v",
			iss.ID, tree.Size(), tree.Depth(), rootHex, time.Since(buildStart))

		issuerDir := filepath.Join(cfg.DataDir, iss.ID)
		rootPath := filepath.Join(issuerDir, "root.json")
		if existingRoot, err := readExistingRoot(rootPath); err == nil {
			if existingRoot == rootHex {
				log.Printf("[%s] Root unchanged, skipping snapshot export", iss.ID)
				continue
			}
		}

		snapshotPath := filepath.Join(issuerDir, "tree-snapshot.json.gz")
		if err := snapshot.ExportFile(tree, parsed.CRLNumber.Uint64(), snapshotPath); err != nil {
			log.Printf("[%s] Skipping: export snapshot: %v", iss.ID, err)
			continue
		}
		log.Printf("[%s] Snapshot exported to %s", iss.ID, snapshotPath)

		if *exportBinary {
			binaryPath := filepath.Join(issuerDir, "tree-snapshot.bin.gz")
			if err := snapshot.ExportBinaryFile(tree, parsed.CRLNumber.Uint64(), binaryPath); err != nil {
				log.Printf("[%s] Warning: binary export failed: %v", iss.ID, err)
			} else {
				log.Printf("[%s] Binary snapshot exported to %s", iss.ID, binaryPath)
			}
		}

		info := rootInfo{
			Root:      rootHex,
			Size:      tree.Size(),
			LeafCount: tree.LeafCount(),
			Depth:     tree.Depth(),
			CRLNumber: parsed.CRLNumber.String(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		rootJSON, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Printf("[%s] Skipping: marshal root.json: %v", iss.ID, err)
			continue
		}
		if err := os.WriteFile(rootPath, rootJSON, 0o644); err != nil {
			log.Printf("[%s] Skipping: write root.json: %v", iss.ID, err)
			continue
		}
		log.Printf("[%s] Root info written to %s", iss.ID, rootPath)

		anyChanged = true
	}

	if ghOutput := os.Getenv("GITHUB_OUTPUT"); ghOutput != "" {
		f, err := os.OpenFile(ghOutput, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatalf("Failed to open GITHUB_OUTPUT: %v", err)
		}
		defer f.Close()
		fmt.Fprintf(f, "changed=%t\n", anyChanged)
	}

	if anyChanged {
		log.Println("Done — LeanIMT+ data updated")
	} else {
		log.Println("Done — no changes")
	}
}

func postRootOnChain(cfg *config.Config) {
	if cfg.RPCURL == "" || cfg.RelayerPrivateKey == "" || cfg.ContractAddress == "" {
		log.Println("Skipping on-chain posting: RPC_URL, RELAYER_PRIVATE_KEY, or CONTRACT_ADDRESS not set")
		return
	}

	client, err := chain.NewClient(cfg.RPCURL)
	if err != nil {
		log.Fatalf("Failed to connect to RPC: %v", err)
	}
	defer client.Close()

	relayer, err := chain.NewRelayer(client, cfg.RelayerPrivateKey, cfg.ContractAddress)
	if err != nil {
		log.Fatalf("Failed to create relayer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := relayer.VerifyContract(ctx); err != nil {
		cancel()
		log.Fatalf("Contract verification failed: %v", err)
	}
	cancel()
	log.Printf("Contract verified at %s (relayer: %s)", cfg.ContractAddress, relayer.Address().Hex())

	type issuerEntry struct {
		ID       string
		IssuerID [32]byte
	}
	entries := []issuerEntry{
		{ID: "g2", IssuerID: chain.IssuerG2},
		{ID: "g3", IssuerID: chain.IssuerG3},
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	for _, iss := range entries {
		rootPath := filepath.Join(cfg.DataDir, iss.ID, "root.json")
		data, err := os.ReadFile(rootPath)
		if err != nil {
			log.Printf("[%s] Skipping: %v", iss.ID, err)
			continue
		}

		var info rootInfo
		if err := json.Unmarshal(data, &info); err != nil {
			log.Printf("[%s] Skipping: parse root.json: %v", iss.ID, err)
			continue
		}

		root, ok := new(big.Int).SetString(strings.TrimPrefix(info.Root, "0x"), 16)
		if !ok {
			log.Printf("[%s] Skipping: invalid root hex: %s", iss.ID, info.Root)
			continue
		}

		crlNumber, ok := new(big.Int).SetString(info.CRLNumber, 10)
		if !ok {
			log.Printf("[%s] Skipping: invalid crlNumber: %s", iss.ID, info.CRLNumber)
			continue
		}

		if info.Depth < 0 || info.Depth > 255 {
			log.Printf("[%s] Skipping: depth %d out of uint8 range", iss.ID, info.Depth)
			continue
		}
		if info.LeafCount < 0 {
			log.Printf("[%s] Skipping: negative leafCount %d", iss.ID, info.LeafCount)
			continue
		}

		log.Printf("[%s] Posting root on-chain: root=%s crlNumber=%s depth=%d leafCount=%d",
			iss.ID, info.Root, info.CRLNumber, info.Depth, info.LeafCount)
		tx, err := relayer.PostRoot(ctx, iss.IssuerID, root, crlNumber, uint8(info.Depth), uint64(info.LeafCount))
		if err != nil {
			if strings.Contains(err.Error(), "stale CRL") {
				log.Printf("[%s] Already posted (stale CRL), skipping", iss.ID)
				continue
			}
			log.Printf("[%s] Failed to post root: %v", iss.ID, err)
			continue
		}
		log.Printf("[%s] Root posted on-chain: tx=%s", iss.ID, tx.Hash().Hex())
	}

	log.Println("Done — on-chain posting complete")
}

func convertJSONToBinary(jsonPath string) {
	hasher := leanimt.NewPoseidonHasher()

	log.Printf("Loading JSON snapshot from %s", jsonPath)
	tree, crlNumber, err := snapshot.ImportFile(hasher, jsonPath)
	if err != nil {
		log.Fatalf("Failed to import JSON snapshot: %v", err)
	}
	rootHex := "0x0"
	if r := tree.Root(); r != nil {
		rootHex = r.Text(16)
		if len(rootHex) > 16 {
			rootHex = rootHex[:16]
		}
	}
	log.Printf("Loaded: size=%d depth=%d root=0x%s, CRL#%d", tree.Size(), tree.Depth(), rootHex, crlNumber)

	outPath := strings.TrimSuffix(jsonPath, ".json.gz") + ".bin.gz"
	log.Printf("Exporting binary snapshot to %s", outPath)
	if err := snapshot.ExportBinaryFile(tree, crlNumber, outPath); err != nil {
		log.Fatalf("Failed to export binary: %v", err)
	}

	jsonInfo, _ := os.Stat(jsonPath)
	binInfo, _ := os.Stat(outPath)
	if jsonInfo != nil && binInfo != nil {
		log.Printf("JSON: %d bytes, Binary: %d bytes (%.1f%%)",
			jsonInfo.Size(), binInfo.Size(),
			float64(binInfo.Size())/float64(jsonInfo.Size())*100)
	}
	log.Println("Done")
}

func readExistingRoot(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var info rootInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return "", err
	}
	return info.Root, nil
}
