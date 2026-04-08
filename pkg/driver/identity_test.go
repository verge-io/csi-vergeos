package driver

import (
	"context"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

func TestGetPluginInfo_NAS(t *testing.T) {
	ids := &IdentityService{
		driverName:    NASDriverName,
		driverVersion: DriverVersion,
	}

	resp, err := ids.GetPluginInfo(context.Background(), &csi.GetPluginInfoRequest{})
	if err != nil {
		t.Fatalf("GetPluginInfo failed: %v", err)
	}
	if resp.Name != NASDriverName {
		t.Errorf("Name = %q, want %q", resp.Name, NASDriverName)
	}
	if resp.VendorVersion != DriverVersion {
		t.Errorf("VendorVersion = %q, want %q", resp.VendorVersion, DriverVersion)
	}
}

func TestGetPluginInfo_Block(t *testing.T) {
	ids := &IdentityService{
		driverName:    BlockDriverName,
		driverVersion: DriverVersion,
	}

	resp, err := ids.GetPluginInfo(context.Background(), &csi.GetPluginInfoRequest{})
	if err != nil {
		t.Fatalf("GetPluginInfo failed: %v", err)
	}
	if resp.Name != BlockDriverName {
		t.Errorf("Name = %q, want %q", resp.Name, BlockDriverName)
	}
}

func TestProbe(t *testing.T) {
	ids := &IdentityService{
		driverName:    NASDriverName,
		driverVersion: DriverVersion,
	}

	resp, err := ids.Probe(context.Background(), &csi.ProbeRequest{})
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}
	if resp.Ready == nil || !resp.Ready.Value {
		t.Error("Probe should report ready")
	}
}

func TestGetPluginCapabilities(t *testing.T) {
	ids := &IdentityService{
		driverName:    NASDriverName,
		driverVersion: DriverVersion,
	}

	resp, err := ids.GetPluginCapabilities(context.Background(), &csi.GetPluginCapabilitiesRequest{})
	if err != nil {
		t.Fatalf("GetPluginCapabilities failed: %v", err)
	}

	hasController := false
	for _, cap := range resp.Capabilities {
		if svc := cap.GetService(); svc != nil {
			if svc.Type == csi.PluginCapability_Service_CONTROLLER_SERVICE {
				hasController = true
			}
		}
	}
	if !hasController {
		t.Error("should advertise CONTROLLER_SERVICE capability")
	}
}
