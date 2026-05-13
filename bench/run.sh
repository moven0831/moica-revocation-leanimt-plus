#!/bin/sh
# Orchestrates the real-data benchmark across the LeanIMT+ repo (this repo)
# and the SMT repo (sibling clone at $SMT_REPO_DIR). Runs go test -bench in
# both, captures stdout, and invokes the report generator.
#
# Env vars:
#   SMT_REPO_DIR  Path to a clone of moica-revocation-smt with the matching
#                 bench file applied. Defaults to ../moica-revocation-smt.
#   BENCH_COUNT   Number of -count iterations for fast benches (default 3).
#   BENCH_BUILD_TIME  -benchtime for the heavy Build/HashCount benches.
#                 Default 3x (force exactly 3 builds; G2 takes seconds).
#   BENCH_FAST_TIME   -benchtime for ProofGen/Verify benches. Default 1s.
#   MOICA_BENCH_DER_DIR  If already set, fetch.sh is still invoked but won't
#                 re-download cached DERs.

set -eu

script_dir=$(cd "$(dirname "$0")" && pwd)
repo_dir=$(cd "$script_dir/.." && pwd)
cache_dir="$script_dir/.cache"

smt_repo_dir=${SMT_REPO_DIR:-$repo_dir/../moica-revocation-smt}
bench_count=${BENCH_COUNT:-3}
bench_build_time=${BENCH_BUILD_TIME:-3x}
bench_fast_time=${BENCH_FAST_TIME:-1s}

if [ ! -d "$smt_repo_dir" ]; then
	echo "[run.sh] SMT_REPO_DIR not found: $smt_repo_dir" >&2
	echo "[run.sh] Clone moica-revocation-smt as a sibling:" >&2
	echo "         git clone https://github.com/moven0831/moica-revocation-smt $smt_repo_dir" >&2
	exit 1
fi
smt_repo_dir=$(cd "$smt_repo_dir" && pwd)
if [ ! -f "$smt_repo_dir/server/internal/smt/bench_real_test.go" ]; then
	echo "[run.sh] $smt_repo_dir/server/internal/smt/bench_real_test.go is missing." >&2
	echo "[run.sh] Apply the matching SMT-bench PR (see bench/smt-bench-pr/ in this repo) before running." >&2
	exit 1
fi

"$script_dir/fetch.sh"
export MOICA_BENCH_DER_DIR="$cache_dir"
echo "[run.sh] MOICA_BENCH_DER_DIR=$MOICA_BENCH_DER_DIR"

mkdir -p "$cache_dir"

run_bench() {
	label=$1
	dir=$2
	pkg=$3
	out_heavy="$cache_dir/${label}_heavy.txt"
	out_fast="$cache_dir/${label}_fast.txt"
	out_snap="$cache_dir/${label}_snapshot.txt"

	echo "[run.sh] $label heavy benches -> $out_heavy"
	( cd "$dir" && go test -tags=bench_real -benchmem -run='^$' \
		-bench='Build$|HashCount_Build' -benchtime="$bench_build_time" -count=1 \
		"$pkg" ) | tee "$out_heavy"

	echo "[run.sh] $label fast benches -> $out_fast"
	( cd "$dir" && go test -tags=bench_real -benchmem -run='^$' \
		-bench='ProofGen|^BenchmarkReal_Verify$|HashCount_Verify' \
		-benchtime="$bench_fast_time" -count="$bench_count" \
		"$pkg" ) | tee "$out_fast"

	echo "[run.sh] $label snapshot test -> $out_snap"
	( cd "$dir" && go test -tags=bench_real -run='^TestReal_SnapshotSize$' -v \
		"$pkg" ) | tee "$out_snap"
}

run_bench leanimt_plus "$repo_dir/server" ./internal/leanimt_plus
run_bench smt "$smt_repo_dir/server" ./internal/smt

mod_out="$cache_dir/module_divergence.txt"
(
	echo "## leanimt_plus modules"
	( cd "$repo_dir/server" && go list -m all )
	echo
	echo "## smt modules"
	( cd "$smt_repo_dir/server" && go list -m all )
) > "$mod_out"
echo "[run.sh] module list -> $mod_out"

report_out="$repo_dir/bench/RESULTS.md"
echo "[run.sh] generating $report_out"
( cd "$script_dir" && go run ./cmd/report \
	-leanimt-heavy "$cache_dir/leanimt_plus_heavy.txt" \
	-leanimt-fast "$cache_dir/leanimt_plus_fast.txt" \
	-leanimt-snapshot "$cache_dir/leanimt_plus_snapshot.txt" \
	-smt-heavy "$cache_dir/smt_heavy.txt" \
	-smt-fast "$cache_dir/smt_fast.txt" \
	-smt-snapshot "$cache_dir/smt_snapshot.txt" \
	-modules "$mod_out" \
	-out "$report_out" )

echo "[run.sh] done. open $report_out"
