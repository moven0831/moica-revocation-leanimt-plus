# Real-data benchmark — LeanIMT+ vs SMT

This directory drives a side-by-side performance comparison of `moica-revocation-leanimt-plus` (this repo) against the predecessor [`moica-revocation-smt`](https://github.com/moven0831/moica-revocation-smt) on **live Taiwan MOICA G2 + G3 CRL data** (~393k and ~103k revoked serials respectively).

Each impl has a Go bench file inside its own tree package (`server/internal/leanimt_plus/bench_real_test.go` here; matching file in the SMT repo); the harness here orchestrates both runs and produces [`RESULTS.md`](RESULTS.md).

## Reproduce

```sh
# 1. Clone the SMT repo as a sibling.
git clone https://github.com/moven0831/moica-revocation-smt ../moica-revocation-smt

# 2. Apply the matching SMT-bench PR.
#    See bench/smt-bench-pr/bench_real_test.go for the file that must exist at
#    ../moica-revocation-smt/server/internal/smt/bench_real_test.go.

# 3. Run.
make bench-real

# 4. Open the report.
open bench/RESULTS.md   # macOS
xdg-open bench/RESULTS.md   # linux
```

## Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `SMT_REPO_DIR` | `../moica-revocation-smt` | Path to a clone of the SMT repo with the matching bench file applied. |
| `MOICA_BENCH_DER_DIR` | (set by `fetch.sh`) | Directory both impls read DERs from. Hard-skip in bench if unset — both impls must read the same bytes. |
| `CRL_G2_URL` | `https://moica.nat.gov.tw/repository/MOICA/CRL2/complete.crl` | Override MOICA G2 endpoint. |
| `CRL_G3_URL` | `https://crl-moica.moi.gov.tw/crl/MOICA-G3-complete.crl` | Override MOICA G3 endpoint. |
| `BENCH_COUNT` | `3` | `-count` for fast benches (ProofGen, Verify). |
| `BENCH_BUILD_TIME` | `3x` | `-benchtime` for heavy benches (Build, HashCount_Build). |
| `BENCH_FAST_TIME` | `1s` | `-benchtime` for fast benches. |
| `LEAN_BENCH_TIMEOUT` | `2h` | `go test -timeout` for the LeanIMT+ side. G2 `InsertManySorted` takes ~3 min per build; heavy pass runs 6 G2 builds. |
| `SMT_BENCH_TIMEOUT` | `24h` | `go test -timeout` for the SMT side. G2 `BatchAdd` takes ~2 hours per build; heavy pass with `-benchtime=3x` runs 6 G2 builds. Intentionally very generous so the harness never kills a correctly-running bench. |

## What gets measured

The Go bench files in each repo emit:

- **Build time** — `InsertManySorted` (LeanIMT+) / `BatchAdd` with `value=1` (SMT) over deduped+sorted serials.
- **Proof gen time** — `GenerateProof` / `CreateProof` over `K=1024` random queries (seed `42`), for both membership and non-membership.
- **Proof verify time** — same proofs, via `VerifyProof`.
- **Proof shape** — `siblings/op` and `proofBytes/op` (where `proofBytes` is `json.Marshal` of the proof struct).
- **Memory** — `b.ReportAllocs()` bytes-per-op and allocs-per-op for Build and ProofGen.
- **Hash counts** — dedicated `HashCount_*` benches wrap the hasher with an atomic counter. `Hash2` only on the LeanIMT+ side; `Hash2 + Hash3` on the SMT side.
- **Snapshot size** — `TestReal_SnapshotSize` exports the tree via each repo's `snapshot.ExportFile` and stats the file (gzipped + uncompressed). SMT additionally exports `ExportBinary`.

## Why the env-var contract

Live CRL data changes daily. If LeanIMT+ fetched at 10:00 and SMT fetched at 10:05 and a new revocation landed in between, the comparison is unfair. The harness fetches **once** into `bench/.cache/`, sets `MOICA_BENCH_DER_DIR`, and both Go bench files read identical bytes from that directory. The benches `t.Skip()` loudly if `MOICA_BENCH_DER_DIR` is unset — they never fall back to live HTTP.

## Files

- `fetch.sh` — POSIX sh; downloads G2/G3 CRL DERs to `.cache/` (or `--force` to re-fetch). Idempotent.
- `run.sh` — orchestrator: invokes `fetch.sh`, runs four `go test` passes (heavy + fast + snapshot, per repo), runs the report generator.
- `cmd/report/main.go` — parses `go test -bench` output from both repos plus snapshot test logs; emits `RESULTS.md` with side-by-side tables.
- `smt-bench-pr/` — copy of the bench file that must land in the SMT repo. Apply this in a parallel PR to `moica-revocation-smt`.
- `.cache/` — gitignored. Holds DER fixtures + raw `go test` output between runs.

## CI

A manual GitHub Actions workflow exists at `.github/workflows/bench-real.yml` (`workflow_dispatch` only — no auto runs). Inputs: `smt_ref` (git ref of the SMT repo to check out), `runs` (`-count` override). Uploads `RESULTS.md` + raw `.cache/*.txt` as workflow artifacts.

Not wired into the default CI because:
1. ~25 MB live fetch is slow and would hammer the upstream MOICA endpoints on every push.
2. Shared GitHub runners are noisy — timing numbers are only meaningful on a quiet machine.
3. The SMT repo is a separate module that may not be in sync.
