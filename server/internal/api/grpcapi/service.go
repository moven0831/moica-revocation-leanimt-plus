package grpcapi

import (
	"context"
	"math/big"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/moven0831/moica-revocation-smt/server/internal/hexenc"
	"github.com/moven0831/moica-revocation-smt/server/internal/manager"
	pb "github.com/moven0831/moica-revocation-smt/server/pkg/proto/revocation"
)

type RevocationServer struct {
	pb.UnimplementedRevocationProofServiceServer
	mgr       *manager.TreeManager
	startTime time.Time
}

func NewRevocationServer(mgr *manager.TreeManager) *RevocationServer {
	return &RevocationServer{
		mgr:       mgr,
		startTime: time.Now(),
	}
}

func (s *RevocationServer) GetProof(ctx context.Context, req *pb.GetProofRequest) (*pb.GetProofResponse, error) {
	snHex := strings.TrimPrefix(req.SerialNumber, "0x")
	if len(snHex) == 0 || len(snHex) > 32 {
		return nil, status.Error(codes.InvalidArgument, "invalid serial number")
	}

	sn, ok := new(big.Int).SetString(snHex, 16)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid serial number hex")
	}

	proof, err := s.mgr.GetProof(req.IssuerId, sn)
	if err != nil {
		if strings.Contains(err.Error(), "unknown issuer") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.GetProofResponse{
		IssuerId:     req.IssuerId,
		SerialNumber: "0x" + snHex,
		ProofType:    uint32(proof.ProofType),
		Root:         hexenc.Encode(proof.Root),
		Value:        hexenc.Encode(proof.Value),
		Leaf: &pb.IndexedLeaf{
			Value:     hexenc.Encode(proof.Leaf.Value),
			NextValue: hexenc.Encode(proof.Leaf.NextValue),
		},
		LeafIndex: proof.LeafIndex,
		Siblings:  hexenc.EncodeSlice(proof.Siblings),
	}, nil
}

func (s *RevocationServer) GetStatus(ctx context.Context, req *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
	mgStatus := s.mgr.Status()

	generations := make(map[string]*pb.IssuerStatus, len(mgStatus))
	for id, st := range mgStatus {
		generations[id] = &pb.IssuerStatus{
			Loaded:    st.Loaded,
			Size:      int32(st.Size),
			LeafCount: int32(st.LeafCount),
			Depth:     int32(st.Depth),
			Root:      st.Root,
			CrlNumber: st.CRLNumber,
			LoadedAt:  st.LoadedAt,
		}
	}

	return &pb.GetStatusResponse{
		Generations:   generations,
		UptimeSeconds: time.Since(s.startTime).Seconds(),
	}, nil
}

