package rest

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/leanimt_plus"
	"github.com/moven0831/moica-revocation-leanimt-plus/server/internal/manager"
)

func setupTestServer() (*Handler, *manager.TreeManager) {
	h := leanimt_plus.NewPoseidonHasher()
	mgr := manager.New(h)

	tree := leanimt_plus.New(h)
	serial, _ := new(big.Int).SetString("100048210DD2DF2E128096A9282B5EC5", 16)
	if err := tree.Insert(serial); err != nil {
		panic(err)
	}
	mgr.SetTree("g2", tree, 100)

	handler := NewHandler(mgr)
	return handler, mgr
}

func TestGetProofMembership(t *testing.T) {
	handler, _ := setupTestServer()
	router := handler.Router()

	req := httptest.NewRequest("GET", "/proof/g2/100048210DD2DF2E128096A9282B5EC5", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. Body: %s", w.Code, w.Body.String())
	}

	var resp ProofResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal("json decode:", err)
	}

	if resp.ProofType != int(leanimt_plus.ProofMembership) {
		t.Errorf("proofType: got %d, want %d", resp.ProofType, leanimt_plus.ProofMembership)
	}
	if resp.IssuerID != "g2" {
		t.Errorf("issuerId: got %s, want g2", resp.IssuerID)
	}
	if resp.Leaf.Value != resp.Value {
		t.Errorf("leaf.value=%s, value=%s — should match for membership", resp.Leaf.Value, resp.Value)
	}

	h := leanimt_plus.NewPoseidonHasher()
	ok, err := VerifyProofFromResponse(h, &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("proof verification failed")
	}
}

func TestGetProofNonMembership(t *testing.T) {
	handler, _ := setupTestServer()
	router := handler.Router()

	req := httptest.NewRequest("GET", "/proof/g2/FFFFFFFFFFFF", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}

	var resp ProofResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.ProofType != int(leanimt_plus.ProofNonMembership) {
		t.Errorf("proofType: got %d, want %d", resp.ProofType, leanimt_plus.ProofNonMembership)
	}

	h := leanimt_plus.NewPoseidonHasher()
	ok, err := VerifyProofFromResponse(h, &resp)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("non-membership proof verification failed")
	}
}

func TestGetProofUnknownIssuer(t *testing.T) {
	handler, _ := setupTestServer()
	router := handler.Router()

	req := httptest.NewRequest("GET", "/proof/unknown/ABC123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

func TestGetProofInvalidSerial(t *testing.T) {
	handler, _ := setupTestServer()
	router := handler.Router()

	req := httptest.NewRequest("GET", "/proof/g2/ZZZZ", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestGetStatus(t *testing.T) {
	handler, _ := setupTestServer()
	router := handler.Router()

	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}

	var resp StatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal("json decode:", err)
	}

	g2, ok := resp.Generations["g2"]
	if !ok {
		t.Fatal("g2 not in generations")
	}
	if !g2.Loaded {
		t.Error("g2 should be loaded")
	}
	if g2.Size != 1 {
		t.Errorf("g2 size: got %d, want 1", g2.Size)
	}
	if g2.Depth < 1 {
		t.Errorf("g2 depth: got %d, want >= 1", g2.Depth)
	}
	if g2.CRLNumber != 100 {
		t.Errorf("g2 crlNumber: got %d, want 100", g2.CRLNumber)
	}
}
