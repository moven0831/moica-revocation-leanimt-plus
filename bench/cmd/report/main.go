// Command report joins LeanIMT+ and SMT bench output into bench/RESULTS.md.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	benchLineRe = regexp.MustCompile(`^(Benchmark\S+?)(?:-\d+)?\s+(\d+)\s+(.*)$`)
	tokenRe     = regexp.MustCompile(`([0-9.eE+\-]+)\s+(\S+)`)
	snapshotRe  = regexp.MustCompile(`SNAPSHOT\s+(\S+)\s+gz=(\d+)\s+raw=(\d+)\s+leaves=(\d+)\s+depth=(\d+)(?:\s+format=(\S+))?`)
)

type benchResult struct {
	name    string
	iters   int64
	nsPerOp float64
	bPerOp  float64
	allocs  float64
	metrics map[string]float64
}

type snapshotResult struct {
	dataset string
	format  string
	gz      int64
	raw     int64
	leaves  int64
	depth   int64
}

type impl struct {
	label   string
	benches map[string]*benchResult
	snaps   []snapshotResult
}

func newImpl(label string) *impl {
	return &impl{label: label, benches: map[string]*benchResult{}}
}

func parseBenchOutput(path string, target *impl) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return parseBenchReader(f, target)
}

func parseBenchReader(r io.Reader, target *impl) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if m := benchLineRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			iters, _ := strconv.ParseInt(m[2], 10, 64)
			rest := m[3]
			res := &benchResult{name: name, iters: iters, metrics: map[string]float64{}}
			for _, tm := range tokenRe.FindAllStringSubmatch(rest, -1) {
				val, err := strconv.ParseFloat(tm[1], 64)
				if err != nil {
					continue
				}
				unit := tm[2]
				res.metrics[unit] = val
				switch unit {
				case "ns/op":
					res.nsPerOp = val
				case "B/op":
					res.bPerOp = val
				case "allocs/op":
					res.allocs = val
				}
			}
			target.benches[name] = res
		}
		if m := snapshotRe.FindStringSubmatch(line); m != nil {
			gz, _ := strconv.ParseInt(m[2], 10, 64)
			raw, _ := strconv.ParseInt(m[3], 10, 64)
			leaves, _ := strconv.ParseInt(m[4], 10, 64)
			depth, _ := strconv.ParseInt(m[5], 10, 64)
			format := "json.gz"
			if len(m) >= 7 && m[6] != "" {
				format = m[6]
			}
			target.snaps = append(target.snaps, snapshotResult{
				dataset: m[1],
				format:  format,
				gz:      gz,
				raw:     raw,
				leaves:  leaves,
				depth:   depth,
			})
		}
	}
	return sc.Err()
}

func fmtDuration(ns float64) string {
	if ns == 0 || math.IsNaN(ns) {
		return "—"
	}
	d := time.Duration(ns) * time.Nanosecond
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.2f s", float64(d)/float64(time.Second))
	case d >= time.Millisecond:
		return fmt.Sprintf("%.2f ms", float64(d)/float64(time.Millisecond))
	case d >= time.Microsecond:
		return fmt.Sprintf("%.2f µs", float64(d)/float64(time.Microsecond))
	default:
		return fmt.Sprintf("%.0f ns", ns)
	}
}

func fmtBytes(n float64) string {
	if n == 0 {
		return "—"
	}
	const (
		kB = 1024.0
		mB = 1024.0 * 1024.0
		gB = 1024.0 * 1024.0 * 1024.0
	)
	switch {
	case n >= gB:
		return fmt.Sprintf("%.2f GB", n/gB)
	case n >= mB:
		return fmt.Sprintf("%.2f MB", n/mB)
	case n >= kB:
		return fmt.Sprintf("%.1f KB", n/kB)
	default:
		return fmt.Sprintf("%.0f B", n)
	}
}

func fmtInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	out := []byte{}
	for i, ch := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, ch)
	}
	return string(out)
}

func ratio(a, b float64) string {
	if a == 0 || b == 0 {
		return "—"
	}
	return fmt.Sprintf("%.2fx", a/b)
}

// find returns the first bench whose name contains every `include` token and
// none of the `exclude` tokens. nil if no match.
func find(target *impl, include, exclude []string) *benchResult {
	for name, r := range target.benches {
		ok := true
		for _, t := range include {
			if !strings.Contains(name, t) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		for _, t := range exclude {
			if strings.Contains(name, t) {
				ok = false
				break
			}
		}
		if ok {
			return r
		}
	}
	return nil
}

type ctx struct {
	lean *impl
	smt  *impl
	out  *strings.Builder
}

func (c *ctx) wf(format string, args ...any) {
	fmt.Fprintf(c.out, format, args...)
}

func (c *ctx) sectionBuild() {
	c.wf("## Build (full tree from sorted serials)\n\n")
	c.wf("| Dataset | LeanIMT+ time | SMT time | Speedup (SMT / LeanIMT+) | LeanIMT+ allocs/op | SMT allocs/op |\n")
	c.wf("|---|---|---|---|---|---|\n")
	for _, ds := range []string{"G2", "G3"} {
		l := find(c.lean, []string{"Build", ds}, []string{"HashCount"})
		s := find(c.smt, []string{"Build", ds}, []string{"HashCount"})
		ln, sn, sp := "—", "—", "—"
		la, sa := "—", "—"
		if l != nil {
			ln = fmtDuration(l.nsPerOp)
			la = fmtInt(int64(l.allocs))
		}
		if s != nil {
			sn = fmtDuration(s.nsPerOp)
			sa = fmtInt(int64(s.allocs))
		}
		if l != nil && s != nil {
			sp = ratio(s.nsPerOp, l.nsPerOp)
		}
		c.wf("| %s | %s | %s | %s | %s | %s |\n", ds, ln, sn, sp, la, sa)
	}
	c.wf("\n")
}

func (c *ctx) sectionProofGen() {
	c.wf("## Proof generation\n\n")
	c.wf("| Dataset | Case | LeanIMT+ time | SMT time | LeanIMT+ siblings | SMT siblings | LeanIMT+ bytes | SMT bytes |\n")
	c.wf("|---|---|---|---|---|---|---|---|\n")
	for _, ds := range []string{"G2", "G3"} {
		for _, kind := range []string{"Membership", "NonMembership"} {
			l := find(c.lean, []string{"ProofGen", ds, kind}, nil)
			s := find(c.smt, []string{"ProofGen", ds, kind}, nil)
			lt, st := "—", "—"
			lsib, ssib := "—", "—"
			lb, sb := "—", "—"
			if l != nil {
				lt = fmtDuration(l.nsPerOp)
				if v, ok := l.metrics["siblings/op"]; ok {
					lsib = fmt.Sprintf("%.1f", v)
				}
				if v, ok := l.metrics["proofBytes/op"]; ok {
					lb = fmtBytes(v)
				}
			}
			if s != nil {
				st = fmtDuration(s.nsPerOp)
				if v, ok := s.metrics["siblings/op"]; ok {
					ssib = fmt.Sprintf("%.1f", v)
				}
				if v, ok := s.metrics["proofBytes/op"]; ok {
					sb = fmtBytes(v)
				}
			}
			c.wf("| %s | %s | %s | %s | %s | %s | %s | %s |\n",
				ds, kind, lt, st, lsib, ssib, lb, sb)
		}
	}
	c.wf("\n")
}

func (c *ctx) sectionVerify() {
	c.wf("## Proof verification\n\n")
	c.wf("| Dataset | Case | LeanIMT+ time | SMT time | Speedup (SMT / LeanIMT+) |\n")
	c.wf("|---|---|---|---|---|\n")
	for _, ds := range []string{"G2", "G3"} {
		for _, kind := range []string{"Membership", "NonMembership"} {
			l := find(c.lean, []string{"Verify", ds, kind}, []string{"HashCount"})
			s := find(c.smt, []string{"Verify", ds, kind}, []string{"HashCount"})
			lt, st, sp := "—", "—", "—"
			if l != nil {
				lt = fmtDuration(l.nsPerOp)
			}
			if s != nil {
				st = fmtDuration(s.nsPerOp)
			}
			if l != nil && s != nil {
				sp = ratio(s.nsPerOp, l.nsPerOp)
			}
			c.wf("| %s | %s | %s | %s | %s |\n", ds, kind, lt, st, sp)
		}
	}
	c.wf("\n")
}

func (c *ctx) sectionHashCounts() {
	c.wf("## Hash counts (Poseidon-P256 calls per op)\n\n")
	c.wf("LeanIMT+ uses only `Hash2`; SMT uses `Hash2` (internal nodes) and `Hash3` (leaves). Total = `hash2 + hash3`.\n\n")
	c.wf("| Operation | Dataset | LeanIMT+ hash2 | SMT hash2 | SMT hash3 | SMT total |\n")
	c.wf("|---|---|---|---|---|---|\n")

	row := func(op, ds string, include []string) {
		l := find(c.lean, append([]string{"HashCount"}, include...), nil)
		s := find(c.smt, append([]string{"HashCount"}, include...), nil)
		lh2, sh2, sh3, stotal := "—", "—", "—", "—"
		if l != nil {
			lh2 = fmtInt(int64(l.metrics["hash2/op"]))
		}
		if s != nil {
			h2 := s.metrics["hash2/op"]
			h3 := s.metrics["hash3/op"]
			sh2 = fmtInt(int64(h2))
			sh3 = fmtInt(int64(h3))
			stotal = fmtInt(int64(h2 + h3))
		}
		c.wf("| %s | %s | %s | %s | %s | %s |\n", op, ds, lh2, sh2, sh3, stotal)
	}

	for _, ds := range []string{"G2", "G3"} {
		row("Build", ds, []string{"Build", ds})
	}
	for _, ds := range []string{"G2", "G3"} {
		for _, kind := range []string{"Membership", "NonMembership"} {
			row("Verify "+kind, ds, []string{"Verify", ds, kind})
		}
	}
	c.wf("\n")
}

func (c *ctx) sectionSnapshot() {
	c.wf("## Snapshot size on disk\n\n")
	c.wf("| Dataset | Format | LeanIMT+ gz | LeanIMT+ raw | SMT gz | SMT raw |\n")
	c.wf("|---|---|---|---|---|---|\n")

	type key struct{ ds, fmt string }
	keys := map[key]bool{}
	for _, s := range c.lean.snaps {
		keys[key{s.dataset, s.format}] = true
	}
	for _, s := range c.smt.snaps {
		keys[key{s.dataset, s.format}] = true
	}
	ordered := make([]key, 0, len(keys))
	for k := range keys {
		ordered = append(ordered, k)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].ds != ordered[j].ds {
			return ordered[i].ds < ordered[j].ds
		}
		return ordered[i].fmt < ordered[j].fmt
	})

	findSnap := func(snaps []snapshotResult, k key) *snapshotResult {
		for i := range snaps {
			if snaps[i].dataset == k.ds && snaps[i].format == k.fmt {
				return &snaps[i]
			}
		}
		return nil
	}

	for _, k := range ordered {
		l := findSnap(c.lean.snaps, k)
		s := findSnap(c.smt.snaps, k)
		lgz, lraw, sgz, sraw := "—", "—", "—", "—"
		if l != nil {
			lgz = fmtBytes(float64(l.gz))
			lraw = fmtBytes(float64(l.raw))
		}
		if s != nil {
			sgz = fmtBytes(float64(s.gz))
			sraw = fmtBytes(float64(s.raw))
		}
		c.wf("| %s | %s | %s | %s | %s | %s |\n", k.ds, k.fmt, lgz, lraw, sgz, sraw)
	}
	c.wf("\n")

	c.wf("### Tree shape\n\n")
	c.wf("| Dataset | LeanIMT+ leaves | LeanIMT+ depth | SMT leaves | SMT depth |\n")
	c.wf("|---|---|---|---|---|\n")
	for _, ds := range []string{"G2", "G3"} {
		lLeaves, lDepth, sLeaves, sDepth := "—", "—", "—", "—"
		for _, s := range c.lean.snaps {
			if s.dataset == ds {
				lLeaves = fmtInt(s.leaves)
				lDepth = strconv.FormatInt(s.depth, 10)
				break
			}
		}
		for _, s := range c.smt.snaps {
			if s.dataset == ds {
				sLeaves = fmtInt(s.leaves)
				sDepth = strconv.FormatInt(s.depth, 10)
				break
			}
		}
		c.wf("| %s | %s | %s | %s | %s |\n", ds, lLeaves, lDepth, sLeaves, sDepth)
	}
	c.wf("\n")
}

func (c *ctx) sectionModules(modulesPath string) {
	if modulesPath == "" {
		return
	}
	data, err := os.ReadFile(modulesPath)
	if err != nil {
		c.wf("## Module divergence\n\n_(could not read %s: %v)_\n\n", modulesPath, err)
		return
	}
	leanMods, smtMods := splitModuleLists(string(data))
	diff := diffModules(leanMods, smtMods)
	c.wf("## Module divergence\n\n")
	if len(diff) == 0 {
		c.wf("No version differences between the two `go list -m all` outputs.\n\n")
		return
	}
	c.wf("Modules with different versions between the two repos (timing differences may stem from these):\n\n")
	c.wf("| Module | LeanIMT+ | SMT |\n")
	c.wf("|---|---|---|\n")
	for _, d := range diff {
		c.wf("| `%s` | %s | %s |\n", d.name, d.lean, d.smt)
	}
	c.wf("\n")
}

type modDiff struct{ name, lean, smt string }

func splitModuleLists(s string) (lean, smt map[string]string) {
	lean = map[string]string{}
	smt = map[string]string{}
	cur := lean
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## leanimt_plus") {
			cur = lean
			continue
		}
		if strings.HasPrefix(line, "## smt") {
			cur = smt
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			cur[fields[0]] = fields[1]
		} else if len(fields) == 1 {
			cur[fields[0]] = "(main)"
		}
	}
	return
}

func diffModules(lean, smt map[string]string) []modDiff {
	seen := map[string]bool{}
	out := []modDiff{}
	for name, lv := range lean {
		seen[name] = true
		sv, ok := smt[name]
		if !ok {
			out = append(out, modDiff{name, lv, "(absent)"})
			continue
		}
		if sv != lv {
			out = append(out, modDiff{name, lv, sv})
		}
	}
	for name, sv := range smt {
		if seen[name] {
			continue
		}
		out = append(out, modDiff{name, "(absent)", sv})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func (c *ctx) header() {
	c.wf("# LeanIMT+ vs SMT — real-data benchmark\n\n")
	c.wf("Comparison of [`moica-revocation-leanimt-plus`](https://github.com/moven0831/moica-revocation-leanimt-plus) ")
	c.wf("against the predecessor [`moica-revocation-smt`](https://github.com/moven0831/moica-revocation-smt) ")
	c.wf("on live Taiwan MOICA G2 + G3 CRL data.\n\n")
	c.wf("Generated by `bench/cmd/report` on `%s` (GOOS=%s, GOARCH=%s, NumCPU=%d).\n\n",
		time.Now().UTC().Format(time.RFC3339), runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	c.wf("Reproduce: `make bench-real`. See [bench/README.md](README.md) for the env-var contract.\n\n")
}

func (c *ctx) sectionMethodology() {
	c.wf("## Methodology\n\n")
	c.wf("Both impls benchmark on a single fetch of G2 + G3 CRL DERs staged in `bench/.cache/` (path passed via `MOICA_BENCH_DER_DIR`); the benches refuse to run if it's unset, so neither side falls back to live HTTP. Each repo parses the same DERs, dedupes and sorts the revoked serials, then runs `go test -bench` (`-count=3` for fast benches, `-benchtime=3x` for build / hash-count).\n\n")
	c.wf("Proof gen and verify share a fixed query set of `K=%d` serials per dataset, drawn with seed `42` — membership picks from revoked serials, non-membership picks distinct values guaranteed absent. Both repos consume the same `(seed, K)`, so the workload is bit-identical.\n\n", 1024)

	c.wf("## Metric definitions\n\n")
	c.wf("| Metric | Definition |\n")
	c.wf("|---|---|\n")
	c.wf("| Build time | `ns/op` for `InsertManySorted` (LeanIMT+) / `BatchAdd(value=1)` (SMT) over the full sorted serial list. |\n")
	c.wf("| Proof gen / verify time | `ns/op` averaged across the 1024-query set. |\n")
	c.wf("| siblings/op | Mean siblings array length per proof. LeanIMT+ varies (unpaired-right levels contribute no entry); SMT is fixed at `depth=128`. |\n")
	c.wf("| proofBytes/op | `len(json.Marshal(proof))`. Wire formats differ across repos — compare orders of magnitude, not exact bytes. |\n")
	c.wf("| allocs/op, B/op | `b.ReportAllocs()` allocations and bytes per op. |\n")
	c.wf("| Hash counts | Poseidon-P256 `Hash2` / `Hash3` calls per op, captured in dedicated `HashCount_*` benches that wrap the hasher with an atomic counter (timing benches use the plain hasher to avoid counter overhead). |\n")
	c.wf("| Snapshot size | `os.Stat` of the gzipped (`gz`) and uncompressed (`raw`) file from `snapshot.ExportFile`. SMT additionally exports `ExportBinary` (`bin.gz`). |\n")
	c.wf("| Tree shape | Leaves include the LeanIMT+ sentinel at index 0, so `leaves = serials + 1`. Depth is `ceil(log2(leaves))` for LeanIMT+ and fixed at 128 for SMT. |\n\n")
}

func (c *ctx) footer() {
	c.wf("---\n\n")
	c.wf("### Notes\n\n")
	c.wf("- LeanIMT+ uses dynamic depth and `Hash2` only; SMT uses fixed depth 128 and `Hash3` for leaves.\n")
	c.wf("- Build time is wall-clock for `InsertManySorted` (LeanIMT+) / `BatchAdd` (SMT) over deduped+sorted serials.\n")
	c.wf("- Proof gen/verify times average over %d random queries with fixed seed; both impls use identical query sets.\n", 1024)
	c.wf("- `proofBytes` is `json.Marshal` of the proof struct as-defined in each repo. Wire formats differ, so absolute byte counts are indicative, not normative.\n")
	c.wf("- Hash counts are captured in dedicated benches with an atomic-counting hasher; main timing benches use the plain Poseidon hasher.\n")
}

func main() {
	leanHeavy := flag.String("leanimt-heavy", "", "")
	leanFast := flag.String("leanimt-fast", "", "")
	leanSnap := flag.String("leanimt-snapshot", "", "")
	smtHeavy := flag.String("smt-heavy", "", "")
	smtFast := flag.String("smt-fast", "", "")
	smtSnap := flag.String("smt-snapshot", "", "")
	modules := flag.String("modules", "", "")
	outPath := flag.String("out", "bench/RESULTS.md", "")
	flag.Parse()

	lean := newImpl("leanimt_plus")
	smt := newImpl("smt")

	for _, p := range []string{*leanHeavy, *leanFast, *leanSnap} {
		if p == "" {
			continue
		}
		if err := parseBenchOutput(p, lean); err != nil {
			fmt.Fprintf(os.Stderr, "parse %s: %v\n", p, err)
			os.Exit(1)
		}
	}
	for _, p := range []string{*smtHeavy, *smtFast, *smtSnap} {
		if p == "" {
			continue
		}
		if err := parseBenchOutput(p, smt); err != nil {
			fmt.Fprintf(os.Stderr, "parse %s: %v\n", p, err)
			os.Exit(1)
		}
	}

	c := &ctx{lean: lean, smt: smt, out: &strings.Builder{}}
	c.header()
	c.sectionMethodology()
	c.sectionBuild()
	c.sectionProofGen()
	c.sectionVerify()
	c.sectionHashCounts()
	c.sectionSnapshot()
	c.sectionModules(*modules)
	c.footer()

	if err := os.WriteFile(*outPath, []byte(c.out.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *outPath, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "[report] wrote %s (%d bytes)\n", *outPath, c.out.Len())
}
