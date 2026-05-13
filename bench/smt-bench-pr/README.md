# SMT-side bench file (PR handoff)

The companion file in this directory must be applied to
[`moica-revocation-smt`](https://github.com/moven0831/moica-revocation-smt) as
a separate PR before the harness in `bench/run.sh` can run.

## Apply

```sh
cp bench/smt-bench-pr/bench_real_test.go \
   ../moica-revocation-smt/server/internal/smt/bench_real_test.go

# Verify it builds (no DERs needed — vet works without env vars).
( cd ../moica-revocation-smt/server && go vet -tags=bench_real ./internal/smt/... )
```

Open a PR against `moica-revocation-smt` titled e.g.
`bench(smt): add real-data bench file for LeanIMT+ comparison`.

## What the file does

Mirrors the LeanIMT+ bench at
`server/internal/leanimt_plus/bench_real_test.go`, adapted to SMT's API:

| LeanIMT+ side | SMT side |
|---|---|
| `New(h)` + `InsertManySorted(serials)` | `New(h)` (depth 128) + `BatchAdd(serials, big.NewInt(1))` |
| `GenerateProof(v)` | `CreateProof(key)` |
| `VerifyProof(h, p)` | `VerifyProof(h, p, 128)` |
| `len(p.Siblings)` | `len(p.Siblings)` + 1 if `p.MatchingEntry != nil` |
| `Hash2` only | `Hash2` + `Hash3` (Hash3 for leaves) |
| `snapshot.ExportFile` (JSON gz) | `snapshot.ExportFile` (JSON gz) + `snapshot.ExportBinaryFile` (binary gz) |

The query-generation helpers (`membershipQueries`, `nonMembershipQueries`,
seed `42`, `K=1024`) are byte-identical to the LeanIMT+ side so both impls
measure on the same inputs. **If you change these helpers, change them on
both sides.**
