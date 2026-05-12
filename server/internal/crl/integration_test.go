//go:build integration

package crl

import (
	"math/big"
	"sort"
	"testing"
	"time"

	"github.com/moven0831/moica-revocation-smt/server/internal/leanimt"
)

const (
	crlG2URL = "https://moica.nat.gov.tw/repository/MOICA/CRL2/complete.crl"
	crlG3URL = "https://crl-moica.moi.gov.tw/crl/MOICA-G3-complete.crl"
)

func TestIntegrationG2CRL(t *testing.T) {
	testCRLIntegration(t, "G2", crlG2URL, 100_000)
}

func TestIntegrationG3CRL(t *testing.T) {
	testCRLIntegration(t, "G3", crlG3URL, 50_000)
}

func testCRLIntegration(t *testing.T, name, url string, minSerials int) {
	t.Helper()

	t.Logf("[%s] Fetching CRL from %s", name, url)
	fetchStart := time.Now()
	derBytes, err := FetchDER(url)
	if err != nil {
		t.Skipf("Skipping %s: MOICA server unreachable: %v", name, err)
	}
	t.Logf("[%s] Fetched %d bytes in %v", name, len(derBytes), time.Since(fetchStart))

	parseStart := time.Now()
	parsed, err := ParseDER(derBytes)
	if err != nil {
		t.Fatalf("[%s] ParseDER failed: %v", name, err)
	}
	t.Logf("[%s] Parsed %d revoked serials (CRLNumber=%s) in %v",
		name, len(parsed.RevokedSerials), parsed.CRLNumber, time.Since(parseStart))

	if parsed.CRLNumber == nil {
		t.Fatalf("[%s] CRLNumber is nil", name)
	}
	if len(parsed.RevokedSerials) < minSerials {
		t.Fatalf("[%s] Expected at least %d serials, got %d", name, minSerials, len(parsed.RevokedSerials))
	}

	seen := make(map[string]struct{}, len(parsed.RevokedSerials))
	uniqueSerials := make([]*big.Int, 0, len(parsed.RevokedSerials))
	for _, s := range parsed.RevokedSerials {
		key := s.Text(16)
		if _, dup := seen[key]; !dup {
			seen[key] = struct{}{}
			uniqueSerials = append(uniqueSerials, new(big.Int).Set(s))
		}
	}
	sort.Slice(uniqueSerials, func(i, j int) bool { return uniqueSerials[i].Cmp(uniqueSerials[j]) < 0 })
	t.Logf("[%s] %d unique sorted serials (removed %d duplicates)",
		name, len(uniqueSerials), len(parsed.RevokedSerials)-len(uniqueSerials))

	hasher := leanimt.NewPoseidonHasher()
	tree := leanimt.New(hasher)

	buildStart := time.Now()
	err = tree.InsertManyWithProgress(uniqueSerials, 10_000, func(done, total int) {
		t.Logf("[%s] Inserted %d / %d entries", name, done, total)
	})
	if err != nil {
		t.Fatalf("[%s] InsertManySorted failed: %v", name, err)
	}
	buildDuration := time.Since(buildStart)
	root := tree.Root()
	t.Logf("[%s] LeanIMT+ built: size=%d depth=%d root=0x%s duration=%v",
		name, tree.Size(), tree.Depth(), root.Text(16), buildDuration)

	if tree.Size() < minSerials {
		t.Fatalf("[%s] Expected tree size >= %d, got %d", name, minSerials, tree.Size())
	}
	if root == nil || root.Sign() == 0 {
		t.Fatalf("[%s] Root is nil/zero after build", name)
	}

	memberKey := uniqueSerials[0]
	proofStart := time.Now()
	memberProof, err := tree.GenerateProof(memberKey)
	if err != nil {
		t.Fatalf("[%s] GenerateProof failed: %v", name, err)
	}
	t.Logf("[%s] Membership proof generated in %v", name, time.Since(proofStart))

	if memberProof.ProofType != leanimt.ProofMembership {
		t.Fatalf("[%s] Expected membership proof for 0x%s, got %d",
			name, memberKey.Text(16), memberProof.ProofType)
	}
	if !leanimt.VerifyProof(hasher, memberProof) {
		t.Fatalf("[%s] Membership proof verification failed for 0x%s", name, memberKey.Text(16))
	}

	nonMemberKey := big.NewInt(9999)
	nonMemberProof, err := tree.GenerateProof(nonMemberKey)
	if err != nil {
		t.Fatalf("[%s] GenerateProof non-member failed: %v", name, err)
	}
	if nonMemberProof.ProofType != leanimt.ProofNonMembership {
		t.Fatalf("[%s] Expected non-membership proof, got %d", name, nonMemberProof.ProofType)
	}
	if !leanimt.VerifyProof(hasher, nonMemberProof) {
		t.Fatalf("[%s] Non-membership proof verification failed", name)
	}

	t.Logf("[%s] Integration test passed: %d entries, root=0x%s",
		name, tree.Size(), root.Text(16))
}
