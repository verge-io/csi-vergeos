package driver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NodeService implements the CSI Node server by delegating to a NodeBackend.
type NodeService struct {
	csi.UnimplementedNodeServer
	nodeID  string
	backend NodeBackend
}

func NewNodeService(nodeID string, backend NodeBackend) *NodeService {
	return &NodeService{nodeID: nodeID, backend: backend}
}

func (ns *NodeService) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	if ns.backend == nil {
		return nil, status.Error(codes.Unimplemented, "node backend not configured")
	}
	return ns.backend.NodeStageVolume(ctx, req)
}

func (ns *NodeService) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	if ns.backend == nil {
		return nil, status.Error(codes.Unimplemented, "node backend not configured")
	}
	return ns.backend.NodeUnstageVolume(ctx, req)
}

func (ns *NodeService) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if ns.backend == nil {
		return nil, status.Error(codes.Unimplemented, "node backend not configured")
	}
	return ns.backend.NodePublishVolume(ctx, req)
}

func (ns *NodeService) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if ns.backend == nil {
		return nil, status.Error(codes.Unimplemented, "node backend not configured")
	}
	return ns.backend.NodeUnpublishVolume(ctx, req)
}

func (ns *NodeService) NodeGetInfo(_ context.Context, _ *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: ns.nodeID,
	}, nil
}

func (ns *NodeService) NodeGetCapabilities(_ context.Context, _ *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	if ns.backend == nil {
		return &csi.NodeGetCapabilitiesResponse{}, nil
	}
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: ns.backend.NodeGetCapabilities(),
	}, nil
}
