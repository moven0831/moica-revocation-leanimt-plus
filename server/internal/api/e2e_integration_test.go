//go:build integration

package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/api/grpcapi"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/api/rest"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/leanimt"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/manager"
	pb "github.com/moven0831/moica-revocation-leanimt-plus/server/pkg/proto/revocation"
)

const (
	issuerID   = "g2"
	bufSize    = 1024 * 1024
	syntheticN = 1024
)

var (
	testRouter     http.Handler
	testGRPCClient pb.RevocationProofServiceClient
	grpcCleanup    func()
	hasher         leanimt.Hasher
	memberSerials  []string
)

func TestMain(m *testing.M) {
	hasher = leanimt.NewPoseidonHasher()

	tree, serials := buildSyntheticTree(syntheticN)
	memberSerials = pickMembers(serials, 10)

	mgr := manager.New(hasher)
	mgr.SetTree(issuerID, tree, 100)
	mgr.SetTree("g3", leanimt.New(hasher), 0)

	testRouter = rest.NewHandler(mgr).Router()

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	pb.RegisterRevocationProofServiceServer(srv, grpcapi.NewRevocationServer(mgr))
	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "grpc.NewClient: %v\n", err)
		os.Exit(1)
	}
	testGRPCClient = pb.NewRevocationProofServiceClient(conn)
	grpcCleanup = func() {
		conn.Close()
		srv.Stop()
	}

	code := m.Run()

	grpcCleanup()
	os.Exit(code)
}

func buildSyntheticTree(n int) (*leanimt.LeanIMTPlus, []*big.Int) {
	r := rand.New(rand.NewSource(int64(n) * 31))
	seen := make(map[int64]struct{}, n)
	values := make([]*big.Int, 0, n)
	for len(values) < n {
		v := r.Int63n(1<<60) + 1
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		values = append(values, big.NewInt(v))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Cmp(values[j]) < 0 })

	tree := leanimt.New(leanimt.NewPoseidonHasher())
	if err := tree.InsertManySorted(values); err != nil {
		panic(err)
	}
	return tree, values
}

func pickMembers(serials []*big.Int, count int) []string {
	if count > len(serials) {
		count = len(serials)
	}
	step := len(serials) / count
	if step < 1 {
		step = 1
	}
	out := make([]string, 0, count)
	for i := 0; i < len(serials) && len(out) < count; i += step {
		out = append(out, serials[i].Text(16))
	}
	return out
}

func getProofResponse(t *testing.T, issuer, sn string) rest.ProofResponse {
	t.Helper()
	req := httptest.NewRequest("GET", fmt.Sprintf("/proof/%s/%s", issuer, sn), nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /proof/%s/%s: status %d, body: %s", issuer, sn, w.Code, w.Body.String())
	}

	var resp rest.ProofResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	return resp
}

func getStatusResponse(t *testing.T) rest.StatusResponse {
	t.Helper()
	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /status: status %d", w.Code)
	}

	var resp rest.StatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	return resp
}

func parseHex(t *testing.T, s string) *big.Int {
	t.Helper()
	n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
	if !ok {
		t.Fatalf("failed to parse hex: %s", s)
	}
	return n
}

func TestIntegrationRESTMembershipProof(t *testing.T) {
	for _, serial := range memberSerials {
		t.Run(serial, func(t *testing.T) {
			resp := getProofResponse(t, issuerID, serial)

			if resp.ProofType != int(leanimt.ProofMembership) {
				t.Fatalf("proofType: got %d, want %d", resp.ProofType, leanimt.ProofMembership)
			}
			if resp.Leaf.Value != resp.Value {
				t.Errorf("leaf.value=%s, value=%s — must match for membership", resp.Leaf.Value, resp.Value)
			}
			if len(resp.Siblings) > 20 {
				t.Errorf("siblings length %d unexpectedly large", len(resp.Siblings))
			}

			ok, err := rest.VerifyProofFromResponse(hasher, &resp)
			if err != nil {
				t.Fatalf("VerifyProofFromResponse: %v", err)
			}
			if !ok {
				t.Error("VerifyProofFromResponse failed")
			}
		})
	}
}

func TestIntegrationRESTNonMembershipProof(t *testing.T) {
	nonMembers := []string{"1", "2", "DEAD", "FFFFFFFFFFFFFFFF"}

	for _, serial := range nonMembers {
		t.Run(serial, func(t *testing.T) {
			resp := getProofResponse(t, issuerID, serial)

			if resp.ProofType != int(leanimt.ProofNonMembership) {
				t.Fatalf("proofType: got %d, want %d", resp.ProofType, leanimt.ProofNonMembership)
			}
			leafVal := parseHex(t, resp.Leaf.Value)
			leafNext := parseHex(t, resp.Leaf.NextValue)
			value := parseHex(t, resp.Value)
			if leafVal.Cmp(value) >= 0 {
				t.Errorf("low leaf invariant: leaf.value=%s !< value=%s", leafVal, value)
			}
			if leafNext.Sign() != 0 && leafNext.Cmp(value) <= 0 {
				t.Errorf("low leaf invariant: value=%s !< leaf.nextValue=%s", value, leafNext)
			}

			ok, err := rest.VerifyProofFromResponse(hasher, &resp)
			if err != nil {
				t.Fatalf("VerifyProofFromResponse: %v", err)
			}
			if !ok {
				t.Error("VerifyProofFromResponse failed")
			}
		})
	}
}

func TestIntegrationRESTRootConsistency(t *testing.T) {
	statusResp := getStatusResponse(t)

	g2Status, ok := statusResp.Generations[issuerID]
	if !ok {
		t.Fatal("g2 not in generations")
	}
	if !g2Status.Loaded {
		t.Fatal("g2 not loaded")
	}
	if g2Status.Size != syntheticN {
		t.Errorf("g2 size: got %d, want %d", g2Status.Size, syntheticN)
	}

	proofResp := getProofResponse(t, issuerID, memberSerials[0])
	if proofResp.Root != g2Status.Root {
		t.Errorf("root mismatch: proof=%s, status=%s", proofResp.Root, g2Status.Root)
	}

	nonMemberResp := getProofResponse(t, issuerID, "DEAD")
	if nonMemberResp.Root != g2Status.Root {
		t.Errorf("non-member root mismatch: proof=%s, status=%s", nonMemberResp.Root, g2Status.Root)
	}
}

func TestIntegrationGRPCMembershipProof(t *testing.T) {
	ctx := context.Background()

	for _, serial := range memberSerials {
		t.Run(serial, func(t *testing.T) {
			resp, err := testGRPCClient.GetProof(ctx, &pb.GetProofRequest{
				IssuerId:     issuerID,
				SerialNumber: serial,
			})
			if err != nil {
				t.Fatalf("GetProof: %v", err)
			}
			if resp.ProofType != uint32(leanimt.ProofMembership) {
				t.Fatalf("proofType: got %d, want %d", resp.ProofType, leanimt.ProofMembership)
			}
			if resp.Leaf.Value != resp.Value {
				t.Errorf("leaf.value=%s, value=%s — must match for membership", resp.Leaf.Value, resp.Value)
			}
		})
	}
}

func TestIntegrationGRPCNonMembershipProof(t *testing.T) {
	ctx := context.Background()
	nonMembers := []string{"1", "2", "DEAD"}

	for _, serial := range nonMembers {
		t.Run(serial, func(t *testing.T) {
			resp, err := testGRPCClient.GetProof(ctx, &pb.GetProofRequest{
				IssuerId:     issuerID,
				SerialNumber: serial,
			})
			if err != nil {
				t.Fatalf("GetProof: %v", err)
			}
			if resp.ProofType != uint32(leanimt.ProofNonMembership) {
				t.Fatalf("proofType: got %d, want %d", resp.ProofType, leanimt.ProofNonMembership)
			}
		})
	}
}
