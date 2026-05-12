package chain

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/moven0831/moica-revocation-smt/server/internal/chain/contract"
)

type testEnv struct {
	backend  *simulated.Backend
	key      *ecdsa.PrivateKey
	relayer  *Relayer
	contract *contract.LeanIMTPlusRootStorage
	addr     common.Address
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	relayerAddr := crypto.PubkeyToAddress(key.PublicKey)

	backend := simulated.NewBackend(types.GenesisAlloc{
		relayerAddr: {Balance: new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10))},
	})
	t.Cleanup(func() { backend.Close() })

	client := backend.Client()

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		t.Fatal(err)
	}

	addr, _, instance, err := contract.DeployLeanIMTPlusRootStorage(auth, client, relayerAddr)
	if err != nil {
		t.Fatal(err)
	}
	backend.Commit()

	r := &Relayer{
		backend:         client,
		privateKey:      key,
		contractAddress: addr,
		chainID:         chainID,
	}

	return &testEnv{
		backend:  backend,
		key:      key,
		relayer:  r,
		contract: instance,
		addr:     addr,
	}
}

// commitPeriodically drives the simulated backend forward so WaitMined
// can return. Caller MUST close done and wait on the returned function before
// the backend is torn down, otherwise the goroutine races backend.Close.
func commitPeriodically(t *testing.T, backend *simulated.Backend, done <-chan struct{}) func() {
	t.Helper()
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				backend.Commit()
			}
		}
	}()
	return func() { <-stopped }
}

func TestIssuerIDs(t *testing.T) {
	g2 := crypto.Keccak256Hash([]byte("MOICA-G2"))
	g3 := crypto.Keccak256Hash([]byte("MOICA-G3"))

	if IssuerG2 != g2 {
		t.Errorf("IssuerG2 mismatch: got %x, want %x", IssuerG2, g2)
	}
	if IssuerG3 != g3 {
		t.Errorf("IssuerG3 mismatch: got %x, want %x", IssuerG3, g3)
	}
	if IssuerG2 == IssuerG3 {
		t.Error("IssuerG2 and IssuerG3 should differ")
	}
}

func TestVerifyContract(t *testing.T) {
	env := newTestEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := env.relayer.VerifyContract(ctx); err != nil {
		t.Fatalf("VerifyContract failed: %v", err)
	}
}

func TestVerifyContractWrongAddress(t *testing.T) {
	env := newTestEnv(t)
	env.relayer.contractAddress = common.HexToAddress("0x0000000000000000000000000000000000000001")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := env.relayer.VerifyContract(ctx)
	if err == nil {
		t.Fatal("expected error for wrong contract address, got nil")
	}
}

func TestPostRoot(t *testing.T) {
	env := newTestEnv(t)

	done := make(chan struct{})
	wait := commitPeriodically(t, env.backend, done)
	defer func() { close(done); wait() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	root := big.NewInt(12345)
	crlNumber := big.NewInt(100)
	depth := uint8(19)
	leafCount := uint64(412345)

	tx, err := env.relayer.PostRoot(ctx, IssuerG2, root, crlNumber, depth, leafCount)
	if err != nil {
		t.Fatalf("PostRoot failed: %v", err)
	}
	if tx == nil {
		t.Fatal("expected non-nil transaction")
	}

	storedRoot, err := env.contract.GetRoot(nil, IssuerG2)
	if err != nil {
		t.Fatal(err)
	}
	if storedRoot.Cmp(root) != 0 {
		t.Errorf("stored root = %s, want %s", storedRoot, root)
	}

	info, err := env.contract.GetRootInfo(nil, IssuerG2)
	if err != nil {
		t.Fatal(err)
	}
	if info.Depth != depth {
		t.Errorf("stored depth = %d, want %d", info.Depth, depth)
	}
	if info.LeafCount != leafCount {
		t.Errorf("stored leafCount = %d, want %d", info.LeafCount, leafCount)
	}
}

func TestPostRootStaleCRL(t *testing.T) {
	env := newTestEnv(t)

	done := make(chan struct{})
	wait := commitPeriodically(t, env.backend, done)
	defer func() { close(done); wait() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := env.relayer.PostRoot(ctx, IssuerG2, big.NewInt(111), big.NewInt(100), 5, 32)
	if err != nil {
		t.Fatal(err)
	}

	_, err = env.relayer.PostRoot(ctx, IssuerG2, big.NewInt(222), big.NewInt(100), 5, 32)
	if err == nil {
		t.Fatal("expected error for stale CRL number, got nil")
	}
	if !strings.Contains(err.Error(), "stale CRL") {
		t.Errorf("expected 'stale CRL' in error, got: %v", err)
	}
}

func TestPostRootMultipleIssuers(t *testing.T) {
	env := newTestEnv(t)

	done := make(chan struct{})
	wait := commitPeriodically(t, env.backend, done)
	defer func() { close(done); wait() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := env.relayer.PostRoot(ctx, IssuerG2, big.NewInt(111), big.NewInt(10), 4, 16)
	if err != nil {
		t.Fatal(err)
	}

	_, err = env.relayer.PostRoot(ctx, IssuerG3, big.NewInt(222), big.NewInt(20), 5, 32)
	if err != nil {
		t.Fatal(err)
	}

	g2Root, _ := env.contract.GetRoot(nil, IssuerG2)
	g3Root, _ := env.contract.GetRoot(nil, IssuerG3)

	if g2Root.Cmp(big.NewInt(111)) != 0 {
		t.Errorf("G2 root = %s, want 111", g2Root)
	}
	if g3Root.Cmp(big.NewInt(222)) != 0 {
		t.Errorf("G3 root = %s, want 222", g3Root)
	}
}
