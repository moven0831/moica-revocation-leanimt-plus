#!/bin/sh
# Downloads MOICA G2 / G3 CRL DERs into bench/.cache/ for use by the LeanIMT+
# vs SMT benchmark harness. The same bytes are consumed by both repos via the
# MOICA_BENCH_DER_DIR env var to guarantee identical input.
#
# Usage:
#   ./bench/fetch.sh           # fetch if missing
#   ./bench/fetch.sh --force   # re-fetch even if cached
#
# Env vars:
#   CRL_G2_URL, CRL_G3_URL — override default MOICA endpoints
#
# On success, prints `MOICA_BENCH_DER_DIR=<abs path>` on its last line as a
# convenience for standalone use (eval is not needed — run.sh sets the path
# directly from $script_dir/.cache).

set -eu

force=0
for arg in "$@"; do
	case "$arg" in
		--force|-f) force=1 ;;
		-h|--help)
			sed -n '2,16p' "$0"
			exit 0
			;;
		*)
			echo "unknown arg: $arg" >&2
			exit 2
			;;
	esac
done

# Resolve script dir so this works from any cwd.
script_dir=$(cd "$(dirname "$0")" && pwd)
cache_dir="$script_dir/.cache"
mkdir -p "$cache_dir"

g2_url=${CRL_G2_URL:-https://moica.nat.gov.tw/repository/MOICA/CRL2/complete.crl}
g3_url=${CRL_G3_URL:-https://crl-moica.moi.gov.tw/crl/MOICA-G3-complete.crl}

fetch_one() {
	name=$1
	url=$2
	out="$cache_dir/$name.crl.der"
	if [ "$force" -eq 0 ] && [ -s "$out" ]; then
		size=$(wc -c < "$out" | tr -d ' ')
		echo "[fetch.sh] $name: cached ($size bytes) $out" >&2
		return 0
	fi
	echo "[fetch.sh] $name: GET $url" >&2
	tmp="$out.tmp.$$"
	if ! curl -fsSL --max-time 120 -o "$tmp" -- "$url"; then
		rm -f "$tmp"
		echo "[fetch.sh] $name: download failed" >&2
		return 1
	fi
	mv "$tmp" "$out"
	size=$(wc -c < "$out" | tr -d ' ')
	echo "[fetch.sh] $name: wrote $size bytes" >&2
}

fetch_one g2 "$g2_url" &
g2_pid=$!
fetch_one g3 "$g3_url" &
g3_pid=$!
wait "$g2_pid"
wait "$g3_pid"

if command -v shasum >/dev/null 2>&1; then
	shasum -a 256 "$cache_dir"/*.crl.der >&2 || true
elif command -v sha256sum >/dev/null 2>&1; then
	sha256sum "$cache_dir"/*.crl.der >&2 || true
fi

echo "MOICA_BENCH_DER_DIR=$cache_dir"
