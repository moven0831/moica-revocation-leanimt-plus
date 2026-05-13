# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go+Solidity pipeline that fetches Taiwan MOICA Certificate Revocation Lists (CRLs), builds a [**LeanIMT+**](https://github.com/vplasencia/leanimt-plus) (indexed Merkle tree with native non-membership proofs) from revoked serial numbers, serves ZK-friendly membership/non-membership proofs via REST and gRPC, and posts roots on-chain. Forked from `moica-revocation-smt`; the original SMT (which was `@zk-kit/smt` wire-compatible) has been replaced.

## Build & Test Commands

### Go Server (run from `server/`)

```bash
make build              # Compile bin/leanimt-plus-server
make build-cli          # Compile bin/leanimt-plus-build
make test               # Unit tests (excludes integration)
make test-integration   # API E2E (synthetic tree, ~1s) + live CRL fetch (~30 min)
make proto              # Regenerate gRPC stubs from .proto
make run                # Build and run leanimt-plus-server

# Single test
go test ./internal/leanimt_plus -run TestMembership -v
```

### Solidity Contracts (run from `onchain-contract/`)

```bash
source ~/.nvm/nvm.sh && nvm use 22  # Hardhat 3 requires Node >= 22.10.0
pnpm install
npx hardhat test
```

## Architecture

Two entry points in `server/cmd/`:
- **leanimt-plus-server** — Long-running REST (port 3000) + gRPC (port 50051) server with CRL polling
- **leanimt-plus-build** — One-shot CLI: fetch CRL → build LeanIMT+ → export snapshot (used in CI cron). With `--post-root`, reads `root.json` files and posts roots on-chain via `LeanIMTPlusRootStorage.setRoot()`

### Key packages (`server/internal/`)

| Package | Purpose |
|---------|---------|
| `leanimt_plus/` | LeanIMT+ core: Poseidon-P256 hash, dynamic-depth tree, sorted indexed leaves, membership/non-membership proofs |
| `crl/` | CRL HTTP fetcher, DER parser, periodic watcher goroutine |
| `manager/` | Thread-safe per-issuer (`g2`, `g3`) tree management via `TreeManager` |
| `api/rest/` | Chi router: `GET /proof/{issuerId}/{sn}`, `GET /status` |
| `api/grpcapi/` | gRPC mirror of REST API |
| `chain/` | Ethereum client wrapper + relayer for `setRoot` transactions. `chain/contract/` has abigen-generated bindings (`LeanIMTPlusRootStorage`) |
| `snapshot/` | Gzip JSON v2 export/import + GitHub Release download |
| `store/` | Store interface with MemoryStore and BadgerStore implementations |
| `config/` | Environment variable loader |

### Startup flow (leanimt-plus-server)

1. Load config from env vars
2. Import snapshots: local file → GitHub Release fallback → live CRL rebuild
3. Start CRL watcher (polls every 6h; calls `InsertManySorted` on pre-sorted serials, then atomic `SetTree`)
4. Start REST + gRPC servers
5. Graceful shutdown on SIGINT/SIGTERM

### Smart Contract (`onchain-contract/contracts/LeanIMTPlusRootStorage.sol`)

Root registry with tree metadata: `setRoot(bytes32 issuerId, uint256 newRoot, uint256 crlNumber, uint8 depth, uint64 leafCount)` with relayer-only access and monotonic CRL number enforcement. Issuer IDs are `keccak256("MOICA-G2")` / `keccak256("MOICA-G3")`. Emits `RootUpdated(issuerId, root, crlNumber, depth, leafCount)`.

## LeanIMT+ Implementation Details

- **Hash:** Poseidon over P-256 base field via `go-poseidon-p256` (Hash2 only — no Hash3 needed)
- **Depth:** dynamic, `ceil(log2(leafCount))`
- **Leaf commitment:** `Hash2(value, nextValue)`. Leaves form an implicit sorted linked list keyed by `value`.
- **Branch:** `Hash2(left, right)`. Unpaired right children are promoted unchanged (no zero-padding).
- **Sentinel:** first insert creates `{value: 0, nextValue: smallest}` at index 0. The sentinel is never reported by `IndexOf` and `0` is rejected as insertable.
- **Proof shape:** `{proofType, root, value, leaf:{value,nextValue}, leafIndex, siblings}`. `proofType=0` for membership (`leaf.value == value`), `proofType=1` for non-membership (`leaf.value < value < leaf.nextValue` or `leaf.nextValue == 0` tail). `leafIndex` packs path bits LSB-first; bit `i` selects direction for `siblings[i]`. Siblings array length is variable — unpaired-right levels contribute no entry.
- **Insertion order:** production paths pre-sort and use `InsertManySorted` (O(n) total). The naive `InsertMany` is O(n²) and only used in tests.
- **Deletion:** not supported. The watcher rebuilds from scratch every poll.
- **Thread safety:** RWMutex on all public methods.

## CI/CD

- **ci.yml** — On push/PR: Go unit tests + build, Hardhat contract tests, E2E integration tests (synthetic 1024-leaf tree, no network dependency)
- **update-tree.yml** — Cron (04:00 & 16:00 UTC): leanimt-plus-build → commit snapshots → upload to `snapshot-latest` release → `leanimt-plus-build --post-root` posts roots on-chain (Arbitrum Sepolia)

## Data Scale

| CRL | Revoked Certs | DER Size |
|-----|---------------|----------|
| G2  | ~412,000      | ~20MB    |
| G3  | ~103,000      | ~5MB     |

Integration tests use `//go:build integration` tag (excluded from `make test`):
- **API E2E** (`api/e2e_integration_test.go`): Builds a synthetic 1024-leaf tree, verifies proofs via REST + gRPC (~1s, runs in CI)
- **CRL** (`crl/integration_test.go`): Fetches live CRL data and rebuilds trees (~30 min, not in CI)
