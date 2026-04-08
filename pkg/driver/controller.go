package driver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ControllerService implements the CSI Controller server by delegating to a ControllerBackend.
type ControllerService struct {
	csi.UnimplementedControllerServer
	backend ControllerBackend
}

func NewControllerService(backend ControllerBackend) *ControllerService {
	return &ControllerService{backend: backend}
}

func (cs *ControllerService) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	if cs.backend == nil {
		return nil, status.Error(codes.Unimplemented, "controller backend not configured")
	}
	return cs.backend.CreateVolume(ctx, req)
}

func (cs *ControllerService) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if cs.backend == nil {
		return nil, status.Error(codes.Unimplemented, "controller backend not configured")
	}
	return cs.backend.DeleteVolume(ctx, req)
}

func (cs *ControllerService) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	if cs.backend == nil {
		return nil, status.Error(codes.Unimplemented, "controller backend not configured")
	}
	return cs.backend.ControllerPublishVolume(ctx, req)
}

func (cs *ControllerService) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	if cs.backend == nil {
		return nil, status.Error(codes.Unimplemented, "controller backend not configured")
	}
	return cs.backend.ControllerUnpublishVolume(ctx, req)
}

func (cs *ControllerService) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	if cs.backend == nil {
		return nil, status.Error(codes.Unimplemented, "controller backend not configured")
	}
	return cs.backend.ValidateVolumeCapabilities(ctx, req)
}

func (cs *ControllerService) ControllerGetCapabilities(_ context.Context, _ *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	if cs.backend == nil {
		return &csi.ControllerGetCapabilitiesResponse{}, nil
	}
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.backend.ControllerGetCapabilities(),
	}, nil
}
