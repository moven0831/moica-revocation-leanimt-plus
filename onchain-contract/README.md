# LeanIMTPlusRootStorage

On-chain registry for LeanIMT+ roots from MOICA Certificate Revocation Lists.

The CRL pipeline (fetch → parse → build LeanIMT+) produces a Merkle root, depth, and leaf count per issuer, which the CI relayer posts on-chain via `setRoot()`. The contract enforces monotonic CRL numbers to prevent stale updates. Anyone can read the latest root with `getRoot(issuerId)` or full metadata via `getRootInfo(issuerId)` and verify membership/non-membership proofs off-chain.

> **Migration note:** This contract replaces the prior `SMTRootStorage`. The `setRoot` ABI gained `depth` (uint8) and `leafCount` (uint64), and the `RootUpdated` event topic-0 hash changed. Off-chain indexers relying on the old event must re-index. The deployment address from the SMT-era is **not** reusable — redeploy and update `CONTRACT_ADDRESS`.

## Contract

**LeanIMTPlusRootStorage.sol** (Solidity 0.8.28)

| Function | Description |
|----------|-------------|
| `constructor(address _relayer)` | Sets the authorized relayer |
| `setRoot(bytes32 issuerId, uint256 newRoot, uint256 crlNumber, uint8 depth, uint64 leafCount)` | Update root (relayer only) |
| `getRoot(bytes32 issuerId) → uint256` | Read current root |
| `getRootInfo(bytes32 issuerId) → (root, crlNumber, updatedAt, depth, leafCount)` | Read full RootInfo |

**State:**
```solidity
address public relayer;
mapping(bytes32 => RootInfo) public roots;

struct RootInfo {
    uint256 root;
    uint256 crlNumber;
    uint256 updatedAt;
    uint8   depth;
    uint64  leafCount;
}
```

**Events:**
```solidity
event RootUpdated(bytes32 indexed issuerId, uint256 root, uint256 crlNumber, uint8 depth, uint64 leafCount);
```

**Modifiers:**
- `onlyRelayer` — reverts with `"unauthorized"` for non-relayer callers
- `setRoot` requires `crlNumber > roots[issuerId].crlNumber` (monotonic)

## Issuer IDs

| Issuer | ID |
|--------|----|
| MOICA G2 | `keccak256("MOICA-G2")` |
| MOICA G3 | `keccak256("MOICA-G3")` |

## Setup

```bash
nvm use 22    # Hardhat 3 requires Node >= 22.10.0
pnpm install
```

## Test

```bash
npx hardhat test
```

## Deploy

### 1. Generate relayer keypair

```bash
cast wallet new
```

Save the private key (without `0x` prefix) and note the address.

### 2. Fund the relayer

Send Arbitrum Sepolia ETH to the relayer address. You can get testnet ETH from an [Arbitrum Sepolia faucet](https://www.alchemy.com/faucets/arbitrum-sepolia).

### 3. Deploy contract

Local/default network:
```bash
npx hardhat ignition deploy ignition/modules/LeanIMTPlusRootStorage.ts \
  --parameters '{"LeanIMTPlusRootStorageModule": {"relayer": "0x<RELAYER_ADDRESS>"}}'
```

Arbitrum Sepolia (requires `ARB_SEPOLIA_RPC_URL` and `ARB_SEPOLIA_PRIVATE_KEY` env vars):
```bash
npx hardhat ignition deploy ignition/modules/LeanIMTPlusRootStorage.ts \
  --network arbitrumSepolia \
  --parameters '{"LeanIMTPlusRootStorageModule": {"relayer": "0x<RELAYER_ADDRESS>"}}'
```

### 4. Configure GitHub Actions secrets

Set these repository secrets for automated on-chain posting:

| Secret | Value |
|--------|-------|
| `RPC_URL` | Arbitrum Sepolia RPC endpoint (e.g. from Alchemy/Infura) |
| `RELAYER_PRIVATE_KEY` | Hex private key without `0x` prefix |
| `CONTRACT_ADDRESS` | Deployed `LeanIMTPlusRootStorage` contract address |

## CI/CD Integration

The `update-smt.yml` workflow posts roots on-chain automatically via `smtbuild --post-root` after building LeanIMT+ snapshots. It reads `root.json` files and calls `LeanIMTPlusRootStorage.setRoot()` for each issuer. Skips gracefully when secrets are not configured (forks/PRs).
