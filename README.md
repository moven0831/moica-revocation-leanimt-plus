# Moica Revocation LeanIMT+

A Go+Solidity pipeline that turns Taiwan MOICA Certificate Revocation Lists (CRLs) into ZK-friendly membership and non-membership proofs. Revoked serial numbers are committed to a [**LeanIMT+**](https://github.com/vplasencia/leanimt-plus) — an indexed Merkle tree whose sorted-linked-list leaves give cheap native non-membership proofs alongside membership. Roots are served via REST + gRPC and posted on-chain twice daily.

LeanIMT+ replaces the fixed-depth Sparse Merkle Tree that earlier versions of this project used. The savings: dynamic depth (`ceil(log2(n))` instead of a 256-bit key space), variable-length sibling arrays (unpaired-right nodes are promoted, not zero-padded), and `Hash2`-only commitments — no `Hash3` needed.

## Architecture

```
MOICA CRL (DER)
       │
  CRL Fetcher/Parser (internal/crl)
       │
       ▼
  TreeManager (internal/manager) ── per-issuer LeanIMT+ (g2, g3)
       │                                    │
       ▼                                    ▼
  REST API (chi)  +  gRPC API         Chain Relayer
  GET /proof/{issuerId}/{sn}          posts root on-chain
  GET /status                         via LeanIMTPlusRootStorage.sol
```

## Quick Start

```bash
cd server
make build    # → bin/leanimt-plus-server
make run      # starts REST + gRPC servers

# tests
make test

# integration tests (synthetic tree E2E ~1s + live CRL fetch ~30 min)
make test-integration
```

## REST + gRPC API

### Usage Examples

```bash
# Server status (per-issuer size, depth, root, CRL number)
curl localhost:3000/status

# Membership proof for a revoked serial
curl localhost:3000/proof/g2/100048210dd2df2e128096a9282b5ec5

# Non-membership proof for an unrevoked serial
curl localhost:3000/proof/g2/00000000000000000000000000000001
```

### `GET /proof/{issuerId}/{sn}`

Returns a membership (`proofType: 0`) or non-membership (`proofType: 1`) proof. The serial number accepts hex with or without `0x` prefix (max 32 hex chars).

```json
{
  "issuerId": "g2",
  "serialNumber": "0x100048210dd2df2e128096a9282b5ec5",
  "proofType": 1,
  "root": "0x3c2151...",
  "value": "0x100048210dd2df2e128096a9282b5ec5",
  "leaf": {
    "value": "0x0fff48...",
    "nextValue": "0x100050..."
  },
  "leafIndex": 11357,
  "siblings": ["0x...", "0x...", "..."]
}
```

- `proofType` — `0` for membership, `1` for non-membership
- For membership: `leaf.value == value`
- For non-membership: `leaf.value < value < leaf.nextValue` (or `leaf.nextValue == 0` if the low leaf is the tail of the sorted linked list)
- `leafIndex` packs the path bits LSB-first; bit `i` selects the direction of `siblings[i]` during root reconstruction
- `siblings.length` equals the number of levels with a real sibling — LeanIMT+ skips unpaired-right levels, so the array is typically shorter than the nominal tree depth
- All BigInt values are `0x`-prefixed hex strings

### `GET /status`

```json
{
  "generations": {
    "g2": {
      "loaded": true,
      "size": 412404,
      "leafCount": 412405,
      "depth": 19,
      "root": "0x3c2151...",
      "crlNumber": 2026031610,
      "loadedAt": "2026-03-16T08:00:00Z"
    }
  },
  "uptimeSeconds": 3600.5
}
```

`size` is the number of user-inserted leaves (revoked serials). `leafCount` is `size + 1` because LeanIMT+ prepends a sentinel `{value: 0, nextValue: smallest}` at index 0. `depth = ceil(log2(leafCount))`.

### gRPC

Service `RevocationProofService` on port 50051 with `GetProof` and `GetStatus` RPCs. See `server/pkg/proto/revocation/revocation.proto`. The gRPC response mirrors the REST shape (`proofType` / `root` / `value` / `leaf{value,nextValue}` / `leafIndex` / `siblings`).

## Smart Contract

**LeanIMTPlusRootStorage.sol** — on-chain registry for LeanIMT+ roots with tree metadata.

The contract stores each root alongside its `depth` and `leafCount` so verifiers know the expected proof shape for a given root. A CI relayer updates entries twice daily after rebuilding the tree from MOICA CRL data. Each entry is gated on a monotonically increasing CRL number to prevent stale or reorder attacks.

| Function | Description |
|----------|-------------|
| `setRoot(bytes32 issuerId, uint256 newRoot, uint256 crlNumber, uint8 depth, uint64 leafCount)` | Update root (relayer only, monotonic CRL number) |
| `getRoot(bytes32 issuerId) → uint256` | Read current root |
| `getRootInfo(bytes32 issuerId) → (root, crlNumber, updatedAt, depth, leafCount)` | Read full RootInfo |

Event:

```solidity
event RootUpdated(bytes32 indexed issuerId, uint256 root, uint256 crlNumber, uint8 depth, uint64 leafCount);
```

Issuer IDs: `keccak256("MOICA-G2")`, `keccak256("MOICA-G3")`.

### Deploy to Arbitrum Sepolia

1. Generate relayer keypair: `cast wallet new`
2. Fund the relayer with Arbitrum Sepolia ETH
3. Deploy:
   ```bash
   cd onchain-contract
   nvm use 22
   pnpm install
   npx hardhat ignition deploy ignition/modules/LeanIMTPlusRootStorage.ts \
     --network arbitrumSepolia \
     --parameters '{"LeanIMTPlusRootStorageModule": {"relayer": "0x<RELAYER_ADDRESS>"}}'
   ```
4. Set GitHub Actions secrets: `RPC_URL`, `RELAYER_PRIVATE_KEY` (hex, no `0x`), `CONTRACT_ADDRESS`

Post roots on-chain manually (reads `root.json` files; skips gracefully if env vars are unset):

```bash
cd server
./bin/leanimt-plus-build --post-root
```

## Snapshots

Pre-built LeanIMT+ snapshots are published as GitHub Release assets (`snapshot-latest` tag), updated twice daily. On startup the server:

1. Loads local snapshots from `$DATA_DIR/{issuerID}/tree-snapshot.json.gz`
2. Falls back to downloading from the GitHub release
3. Falls back to rebuilding from live CRL data (slow, ~30 min)

Two formats are produced:

- **JSON** (`.json.gz`) — the server's native load format (gzipped v2 schema; see [v2 schema details](server/internal/snapshot/snapshot.go))
- **Binary** (`.bin.gz` / `.bin`) — compact format for client-side WASM loading (see [Client-Side WASM](#client-side-wasm))

Roots and metadata for each cron run are visible in the [snapshot-latest release notes](https://github.com/moven0831/moica-revocation-leanimt-plus/releases/tag/snapshot-latest). LeanIMT+ construction is deterministic given a sorted insertion order, so anyone with the same CRL data can independently rebuild and verify a root.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 3000 | REST API port |
| `GRPC_PORT` | 50051 | gRPC port |
| `DATA_DIR` | `./data` | Storage for CRL data and snapshots |
| `CRL_G2_URL` | MOICA G2 endpoint | CRL download URL for G2 issuer |
| `CRL_G3_URL` | MOICA G3 endpoint | CRL download URL for G3 issuer |
| `CRL_POLL_INTERVAL` | 21600 (6h) | CRL polling interval in seconds |
| `RPC_URL` | — | Ethereum JSON-RPC URL |
| `RELAYER_PRIVATE_KEY` | — | Hex private key for chain relayer |
| `CONTRACT_ADDRESS` | — | `LeanIMTPlusRootStorage` contract address |
| `GITHUB_REPO` | `moven0831/moica-revocation-leanimt-plus` | GitHub repo for snapshot releases |

## Client-Side WASM

A Go WASM module (`leanimt-plus.wasm`) lets clients load the full LeanIMT+ tree and generate proofs entirely in the browser — no server round-trip needed.

### WASM API

Build: `cd server && make build-wasm` (requires Go 1.24+).

The module exposes these functions on `globalThis`:

| Function | Signature | Description |
|----------|-----------|-------------|
| `leanimtPlusLoadSnapshot` | `(Uint8Array)` → `{size, depth, leafCount}` | Load a decompressed v2 binary snapshot in one call |
| `leanimtPlusGenerateProof` | `(valueHex)` → `string` | Generate proof JSON for the given value |
| `leanimtPlusVerifyProof` | `(proofJSON)` → `boolean` | Verify a proof |
| `leanimtPlusRoot` | `()` → `string` | Current tree root (hex) |
| `leanimtPlusGetMemStats` | `()` → `string` | Go runtime memory stats as JSON |

The proof JSON matches the REST `ProofResponse` shape (`proofType`, `root`, `value`, `leaf`, `leafIndex`, `siblings`).

### Binary Snapshot Format (v2)

**Header** (50 bytes, big-endian):

```
[0:2]   magic       uint16  0x4C49 ("LI")
[2:4]   version     uint16  2
[4:6]   depth       uint16
[6:10]  leafCount   uint32
[10:18] crlNumber   uint64
[18:50] root        [32]byte
```

**Per level** (one block for each of `depth+1` levels):

```
count  uint32                  — number of entries at this level
[count repetitions of]
  present uint8                — 0 if hash is absent (nil-promoted), 1 otherwise
  hash    [32]byte             — Poseidon-P256 field element (big-endian)
```

**Per leaf** (one record for each of `leafCount` leaves, including the sentinel at index 0):

```
value     [32]byte
nextValue [32]byte
```

Build binary snapshots: `cd server && make build-binary` or `./bin/leanimt-plus-build --binary`.

Convert an existing JSON snapshot: `./bin/leanimt-plus-build --convert-binary data/g2/tree-snapshot.json.gz`.

## LeanIMT+ Implementation Notes

- **Reference**: [vplasencia/leanimt-plus](https://github.com/vplasencia/leanimt-plus). The Go port lives in `server/internal/leanimt_plus/`.
- **Hash**: Poseidon over the P-256 base field via [`go-poseidon-p256`](https://github.com/zkmopro/go-poseidon-p256). Upstream LeanIMT+ uses Poseidon-BN254, so proofs from this implementation are **not** interoperable with the upstream Circom verifier — we ported the algorithm, not the field.
- **Leaf commitment**: `H(value, nextValue)`. Leaves form an implicit sorted linked list keyed by `value`; `nextValue` is the immediate right neighbor's value.
- **Branch**: `H(left, right)`. Unpaired right children are promoted unchanged at each level (no zero-padding).
- **Sentinel**: the first insert creates `{value: 0, nextValue: smallest}` at index 0. The sentinel is never reported by `IndexOf`, and `0` is rejected as an insertable value.
- **Depth**: dynamic — `ceil(log2(leafCount))`.
- **Insert order**: production paths (`leanimt-plus-build`, CRL watcher) pre-sort serials and use `InsertManySorted` for O(n) total inserts. The naive `InsertMany` is O(n²) and only used in tests.
- **Deletion**: not supported. The CRL watcher rebuilds the tree from scratch every poll, so deletions are unnecessary.

## Data Scale

| CRL | Revoked Certs | File Size |
|-----|--------------|-----------|
| G2  | ~412,000     | ~20 MB DER |
| G3  | ~103,000     | ~5 MB DER  |

For 412k G2 entries, LeanIMT+ depth is `ceil(log2(412k + 1)) = 19`. Membership proofs typically carry ≤ 19 siblings.

## CI/CD

**`ci.yml`** — runs on push/PR to main:

- Go server: `go test ./...` + build binary
- WASM: verify `leanimt-plus.wasm` compiles (`GOOS=js GOARCH=wasm`)
- E2E integration: synthetic tree (1024 leaves) — exercises REST + gRPC + proof verification end-to-end without network
- Contracts: `npx hardhat test` (Node 22)

**`update-tree.yml`** — runs twice daily at 12:00/00:00 UTC+8 (04:00/16:00 UTC):

1. Build server binary + WASM module
2. Fetch CRL, build LeanIMT+, export JSON + binary snapshots (skipped if root unchanged)
3. Upload snapshots, WASM module, and `wasm_exec.js` to GitHub Release (`snapshot-latest`)
4. Post root on-chain via `leanimt-plus-build --post-root` (Arbitrum Sepolia)

Required secrets: `RPC_URL`, `RELAYER_PRIVATE_KEY`, `CONTRACT_ADDRESS`. On-chain posting skips gracefully if any are unset.

## References

- [LeanIMT+](https://github.com/vplasencia/leanimt-plus) — TypeScript reference implementation
- [LeanIMT paper](https://zkkit.org/leanimt-paper.pdf) — base construction
- [Indexed Merkle Tree paper](https://eprint.iacr.org/2021/1263.pdf) — indexed-leaf idea
- [MOICA](https://moica.nat.gov.tw/) — Taiwan citizen digital certificate authority
- [Poseidon Hash](https://www.poseidon-hash.info/) — ZK-friendly hash function

## History

Forked from [`moica-revocation-smt`](https://github.com/moven0831/moica-revocation-smt) in early 2026; the original Sparse Merkle Tree (which was `@zk-kit/smt` wire-compatible) was replaced with LeanIMT+ over Poseidon-P256.
