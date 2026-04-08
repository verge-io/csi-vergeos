//go:build integration

package block

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	vergeos "github.com/verge-io/govergeos"
)

const (
	// testPoolVMID is the VergeOS VM used as the block drive storage pool.
	// Drives are created here before being attached to target VMs.
	testPoolVMID = 40 // testemptyvm01

	// testTargetVMID is the VergeOS VM used as the "node" for publish/unpublish tests.
	testTargetVMID = 41 // testemptyvm02

	// testTargetVMName is the VergeOS VM name used for publish/unpublish tests.
	// This matches the VM at testTargetVMID and is used as the NodeId.
	testTargetVMName = "testemptyvm02"
)

func newIntegrationBlockController(t *testing.T) *BlockController {
	t.Helper()

	for _, env := range []string{"VERGEOS_HOST", "VERGEOS_USERNAME", "VERGEOS_PASSWORD"} {
		if os.Getenv(env) == "" {
			t.Skipf("skipping integration test: %s not set", env)
		}
	}

	// Allow override of pool VM ID via env var.
	poolVMID := testPoolVMID
	if v := os.Getenv("VERGEOS_POOL_VM_ID"); v != "" {
		id, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("invalid VERGEOS_POOL_VM_ID: %v", err)
		}
		poolVMID = id
	}

	client, err := vergeos.NewClient(vergeos.WithEnvConfig())
	if err != nil {
		t.Fatalf("creating VergeOS client: %v", err)
	}

	return NewBlockController(client, poolVMID)
}

func TestIntegration_BlockCreateAndDeleteVolume(t *testing.T) {
	c := newIntegrationBlockController(t)
	ctx := context.Background()

	volumeName := fmt.Sprintf("csi-test-%d", time.Now().UnixNano())

	var volumeID string
	t.Cleanup(func() {
		if volumeID != "" {
			_, err := c.DeleteVolume(ctx, &csi.DeleteVolumeRequest{VolumeId: volumeID})
			if err != nil {
				t.Logf("cleanup: DeleteVolume(%s) failed: %v", volumeID, err)
			}
		}
	})

	// --- CreateVolume ---
	createReq := &csi.CreateVolumeRequest{
		Name: volumeName,
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: 1073741824, // 1 GiB
		},
		VolumeCapabilities: []*csi.VolumeCapability{
			{
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
		Parameters: map[string]string{},
	}

	resp, err := c.CreateVolume(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}
	if resp.Volume == nil {
		t.Fatal("response volume is nil")
	}
	volumeID = resp.Volume.VolumeId
	t.Logf("Created block volume: ID=%s", volumeID)

	// Verify volume context.
	vc := resp.Volume.VolumeContext
	if vc["serial"] == "" {
		t.Error("VolumeContext serial is empty")
	}
	if vc["driveName"] == "" {
		t.Error("VolumeContext driveName is empty")
	}
	t.Logf("Volume context: serial=%s driveName=%s", vc["serial"], vc["driveName"])

	// --- DeleteVolume ---
	_, err = c.DeleteVolume(ctx, &csi.DeleteVolumeRequest{VolumeId: volumeID})
	if err != nil {
		t.Fatalf("DeleteVolume failed: %v", err)
	}
	t.Logf("Deleted block volume: %s", volumeID)
	volumeID = "" // Prevent cleanup from trying to delete again.
}

func TestIntegration_BlockCreateVolume_Idempotent(t *testing.T) {
	c := newIntegrationBlockController(t)
	ctx := context.Background()

	volumeName := fmt.Sprintf("csi-test-%d", time.Now().UnixNano())

	var volumeID string
	t.Cleanup(func() {
		if volumeID != "" {
			_, err := c.DeleteVolume(ctx, &csi.DeleteVolumeRequest{VolumeId: volumeID})
			if err != nil {
				t.Logf("cleanup: DeleteVolume(%s) failed: %v", volumeID, err)
			}
		}
	})

	createReq := &csi.CreateVolumeRequest{
		Name: volumeName,
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: 1073741824,
		},
		VolumeCapabilities: []*csi.VolumeCapability{
			{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}},
		},
		Parameters: map[string]string{},
	}

	// First create.
	resp1, err := c.CreateVolume(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateVolume (first) failed: %v", err)
	}
	volumeID = resp1.Volume.VolumeId
	t.Logf("First CreateVolume returned: %s", volumeID)

	// Second create — idempotent.
	resp2, err := c.CreateVolume(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateVolume (second) failed: %v", err)
	}
	t.Logf("Second CreateVolume returned: %s", resp2.Volume.VolumeId)

	if resp1.Volume.VolumeId != resp2.Volume.VolumeId {
		t.Errorf("idempotent CreateVolume returned different IDs: %q vs %q",
			resp1.Volume.VolumeId, resp2.Volume.VolumeId)
	}
}

func TestIntegration_BlockDeleteVolume_AlreadyGone(t *testing.T) {
	c := newIntegrationBlockController(t)
	ctx := context.Background()

	// Delete a drive that doesn't exist — should succeed (idempotent).
	_, err := c.DeleteVolume(ctx, &csi.DeleteVolumeRequest{VolumeId: "999999"})
	if err != nil {
		t.Fatalf("DeleteVolume of non-existent drive should succeed: %v", err)
	}
	t.Log("DeleteVolume of non-existent drive succeeded (idempotent)")
}

func TestIntegration_BlockPublishAndUnpublish(t *testing.T) {
	c := newIntegrationBlockController(t)
	ctx := context.Background()

	volumeName := fmt.Sprintf("csi-test-%d", time.Now().UnixNano())

	var volumeID string
	t.Cleanup(func() {
		if volumeID != "" {
			// Ensure drive is back on pool VM before deleting.
			_, _ = c.ControllerUnpublishVolume(ctx, &csi.ControllerUnpublishVolumeRequest{
				VolumeId: volumeID,
				NodeId:   "test-node",
			})
			_, err := c.DeleteVolume(ctx, &csi.DeleteVolumeRequest{VolumeId: volumeID})
			if err != nil {
				t.Logf("cleanup: DeleteVolume(%s) failed: %v", volumeID, err)
			}
		}
	})

	// Create a drive on the pool VM.
	createResp, err := c.CreateVolume(ctx, &csi.CreateVolumeRequest{
		Name: volumeName,
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: 1073741824,
		},
		VolumeCapabilities: []*csi.VolumeCapability{
			{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}},
		},
		Parameters: map[string]string{},
	})
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}
	volumeID = createResp.Volume.VolumeId
	t.Logf("Created volume %s for publish test", volumeID)

	// --- ControllerPublishVolume: attach drive to target VM ---
	// NodeId is the VM name; resolveNodeVMID looks it up via the VergeOS API.
	publishResp, err := c.ControllerPublishVolume(ctx, &csi.ControllerPublishVolumeRequest{
		VolumeId: volumeID,
		NodeId:   testTargetVMName,
		VolumeCapability: &csi.VolumeCapability{
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	})
	if err != nil {
		t.Fatalf("ControllerPublishVolume failed: %v", err)
	}
	if publishResp.PublishContext["serial"] == "" {
		t.Error("publish context serial is empty")
	}
	t.Logf("Published volume %s to VM %d (serial=%s)", volumeID, testTargetVMID, publishResp.PublishContext["serial"])

	// Verify drive is now on the target VM.
	driveID, _ := decodeVolumeID(volumeID)
	drive, err := c.vmDrives.Get(ctx, driveID)
	if err != nil {
		t.Fatalf("Get drive after publish failed: %v", err)
	}
	if drive.Machine != testTargetVMID {
		t.Errorf("drive machine = %d, want %d (target VM)", drive.Machine, testTargetVMID)
	}

	// --- ControllerUnpublishVolume: detach drive back to pool VM ---
	_, err = c.ControllerUnpublishVolume(ctx, &csi.ControllerUnpublishVolumeRequest{
		VolumeId: volumeID,
		NodeId:   "test-node",
	})
	if err != nil {
		t.Fatalf("ControllerUnpublishVolume failed: %v", err)
	}
	t.Logf("Unpublished volume %s back to pool VM %d", volumeID, testPoolVMID)

	// Verify drive is back on pool VM.
	drive, err = c.vmDrives.Get(ctx, driveID)
	if err != nil {
		t.Fatalf("Get drive after unpublish failed: %v", err)
	}
	if drive.Machine != testPoolVMID {
		t.Errorf("drive machine = %d, want %d (pool VM)", drive.Machine, testPoolVMID)
	}
}
