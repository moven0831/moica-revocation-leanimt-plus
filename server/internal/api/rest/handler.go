package rest

import (
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/moven0831/moica-revocation-smt/server/internal/hexenc"
	"github.com/moven0831/moica-revocation-smt/server/internal/leanimt"
	"github.com/moven0831/moica-revocation-smt/server/internal/manager"
)

type Handler struct {
	mgr       *manager.TreeManager
	startTime time.Time
}

func NewHandler(mgr *manager.TreeManager) *Handler {
	return &Handler{
		mgr:       mgr,
		startTime: time.Now(),
	}
}

func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/proof/{issuerId}/{sn}", h.getProof)
	r.Get("/status", h.getStatus)

	return r
}

type LeafShape struct {
	Value     string `json:"value"`
	NextValue string `json:"nextValue"`
}

type ProofResponse struct {
	IssuerID     string    `json:"issuerId"`
	SerialNumber string    `json:"serialNumber"`
	ProofType    int       `json:"proofType"`
	Root         string    `json:"root"`
	Value        string    `json:"value"`
	Leaf         LeafShape `json:"leaf"`
	LeafIndex    uint64    `json:"leafIndex"`
	Siblings     []string  `json:"siblings"`
}

type StatusResponse struct {
	Generations   map[string]manager.IssuerStatus `json:"generations"`
	UptimeSeconds float64                         `json:"uptimeSeconds"`
}

func (h *Handler) getProof(w http.ResponseWriter, r *http.Request) {
	issuerID := chi.URLParam(r, "issuerId")
	snHex := chi.URLParam(r, "sn")

	snHex = strings.TrimPrefix(snHex, "0x")
	if len(snHex) == 0 || len(snHex) > 32 {
		http.Error(w, `{"error":"invalid serial number"}`, http.StatusBadRequest)
		return
	}
	sn, ok := new(big.Int).SetString(snHex, 16)
	if !ok {
		http.Error(w, `{"error":"invalid serial number hex"}`, http.StatusBadRequest)
		return
	}

	proof, err := h.mgr.GetProof(issuerID, sn)
	if err != nil {
		if strings.Contains(err.Error(), "unknown issuer") {
			http.Error(w, `{"error":"unknown issuer"}`, http.StatusNotFound)
		} else if err == leanimt.ErrEmptyTree {
			http.Error(w, `{"error":"tree is empty"}`, http.StatusServiceUnavailable)
		} else {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		}
		return
	}

	resp := ProofResponse{
		IssuerID:     issuerID,
		SerialNumber: "0x" + snHex,
		ProofType:    int(proof.ProofType),
		Root:         hexenc.Encode(proof.Root),
		Value:        hexenc.Encode(proof.Value),
		Leaf: LeafShape{
			Value:     hexenc.Encode(proof.Leaf.Value),
			NextValue: hexenc.Encode(proof.Leaf.NextValue),
		},
		LeafIndex: proof.LeafIndex,
		Siblings:  hexenc.EncodeSlice(proof.Siblings),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) getStatus(w http.ResponseWriter, r *http.Request) {
	resp := StatusResponse{
		Generations:   h.mgr.Status(),
		UptimeSeconds: time.Since(h.startTime).Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// VerifyProofFromResponse reconstructs a leanimt.Proof from the wire response
// and runs full verification. Shared with the gRPC client and external
// integration tests so the wire-shape parsing lives in one place.
func VerifyProofFromResponse(h leanimt.Hasher, resp *ProofResponse) (bool, error) {
	root, err := hexenc.Decode(resp.Root)
	if err != nil {
		return false, err
	}
	value, err := hexenc.Decode(resp.Value)
	if err != nil {
		return false, err
	}
	leafVal, err := hexenc.Decode(resp.Leaf.Value)
	if err != nil {
		return false, err
	}
	leafNext, err := hexenc.Decode(resp.Leaf.NextValue)
	if err != nil {
		return false, err
	}
	siblings := make([]*big.Int, len(resp.Siblings))
	for i, s := range resp.Siblings {
		n, err := hexenc.Decode(s)
		if err != nil {
			return false, err
		}
		siblings[i] = n
	}

	p := &leanimt.Proof{
		ProofType: leanimt.ProofType(resp.ProofType),
		Root:      root,
		Value:     value,
		Leaf:      leanimt.IndexedLeaf{Value: leafVal, NextValue: leafNext},
		LeafIndex: resp.LeafIndex,
		Siblings:  siblings,
	}
	return leanimt.VerifyProof(h, p), nil
}
