package driver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// IdentityService implements the CSI Identity server.
type IdentityService struct {
	csi.UnimplementedIdentityServer
	driverName    string
	driverVersion string
}

func NewIdentityService(name, version string) *IdentityService {
	return &IdentityService{
		driverName:    name,
		driverVersion: version,
	}
}

func (ids *IdentityService) GetPluginInfo(_ context.Context, _ *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	return &csi.GetPluginInfoResponse{
		Name:          ids.driverName,
		VendorVersion: ids.driverVersion,
	}, nil
}

func (ids *IdentityService) GetPluginCapabilities(_ context.Context, _ *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}, nil
}

func (ids *IdentityService) Probe(_ context.Context, _ *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{
		Ready: &wrapperspb.BoolValue{Value: true},
	}, nil
}
