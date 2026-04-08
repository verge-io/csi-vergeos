package driver

import (
	"context"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestControllerService_NoBackend(t *testing.T) {
	cs := &ControllerService{}

	_, err := cs.CreateVolume(context.Background(), &csi.CreateVolumeRequest{})
	if err == nil {
		t.Fatal("expected error when no backend is set")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unimplemented {
		t.Errorf("expected Unimplemented, got %v", err)
	}
}

// mockControllerBackend is a minimal mock for testing delegation.
type mockControllerBackend struct {
	createCalled bool
}

func (m *mockControllerBackend) CreateVolume(_ context.Context, _ *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	m.createCalled = true
	return &csi.CreateVolumeResponse{}, nil
}
func (m *mockControllerBackend) DeleteVolume(_ context.Context, _ *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	return &csi.DeleteVolumeResponse{}, nil
}
func (m *mockControllerBackend) ControllerPublishVolume(_ context.Context, _ *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return &csi.ControllerPublishVolumeResponse{}, nil
}
func (m *mockControllerBackend) ControllerUnpublishVolume(_ context.Context, _ *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}
func (m *mockControllerBackend) ValidateVolumeCapabilities(_ context.Context, _ *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return &csi.ValidateVolumeCapabilitiesResponse{}, nil
}
func (m *mockControllerBackend) ControllerGetCapabilities() []*csi.ControllerServiceCapability {
	return nil
}

func TestControllerService_DelegatesToBackend(t *testing.T) {
	mock := &mockControllerBackend{}
	cs := &ControllerService{backend: mock}

	_, err := cs.CreateVolume(context.Background(), &csi.CreateVolumeRequest{})
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}
	if !mock.createCalled {
		t.Error("expected CreateVolume to delegate to backend")
	}
}
