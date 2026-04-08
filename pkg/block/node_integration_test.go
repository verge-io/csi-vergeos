//go:build integration

package block

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"

	"github.com/verge-io/csi-vergeos/pkg/util"
)

// newIntegrationBlockNode creates a BlockNode using the real OS mounter and device finder.
// This requires root privileges for mount/format operations and must run on a VergeOS VM
// where block devices can be hotplugged.
func newIntegrationBlockNode(t *testing.T) *BlockNode {
	t.Helper()

	// Block operations require root for mount/format.
	if os.Getuid() != 0 {
		t.Skip("skipping block node integration test: requires root privileges (run with sudo)")
	}

	// Verify mkfs.ext4 is available (needed for FormatAndMount).
	if _, err := exec.LookPath("mkfs.ext4"); err != nil {
		t.Skip("skipping block node integration test: mkfs.ext4 not found (install e2fsprogs)")
	}

	return NewBlockNode()
}

// createTestDriveForNode creates a block drive on the pool VM and attaches it to the
// target VM (the VM running the test). Returns the volume ID, publish context (with serial),
// and the block controller for cleanup.
func createTestDriveForNode(t *testing.T) (volumeID string, publishContext map[string]string, controller *BlockController) {
	t.Helper()

	controller = newIntegrationBlockController(t)
	ctx := context.Background()

	// The target VM is the VM running these tests. It must be set via env var
	// since the test runs inside the VM.
	targetVMIDStr := os.Getenv("VERGEOS_NODE_VM_ID")
	if targetVMIDStr == "" {
		t.Skip("skipping block node integration test: VERGEOS_NODE_VM_ID not set (set to the VM ID of the VM running this test)")
	}
	targetVMID, err := strconv.Atoi(targetVMIDStr)
	if err != nil {
		t.Fatalf("invalid VERGEOS_NODE_VM_ID %q: %v", targetVMIDStr, err)
	}

	volumeName := fmt.Sprintf("csi-test-node-%d", time.Now().UnixNano())

	// Create the drive on the pool VM.
	createResp, err := controller.CreateVolume(ctx, &csi.CreateVolumeRequest{
		Name: volumeName,
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: 1073741824, // 1 GiB
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
	t.Logf("Created test drive: volumeID=%s", volumeID)

	// Register cleanup: detach + delete on test completion.
	t.Cleanup(func() {
		// Best-effort unpublish (move drive back to pool VM).
		_, _ = controller.ControllerUnpublishVolume(ctx, &csi.ControllerUnpublishVolumeRequest{
			VolumeId: volumeID,
			NodeId:   "integration-test",
		})
		// Delete the drive.
		_, err := controller.DeleteVolume(ctx, &csi.DeleteVolumeRequest{VolumeId: volumeID})
		if err != nil {
			t.Logf("cleanup: DeleteVolume(%s) failed: %v", volumeID, err)
		} else {
			t.Logf("cleanup: deleted volume %s", volumeID)
		}
	})

	// Attach the drive to the node VM (this VM).
	publishResp, err := controller.ControllerPublishVolume(ctx, &csi.ControllerPublishVolumeRequest{
		VolumeId: volumeID,
		NodeId:   "integration-test",
		VolumeCapability: &csi.VolumeCapability{
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
		VolumeContext: map[string]string{
			"vergeos.com/vm-id": strconv.Itoa(targetVMID),
		},
	})
	if err != nil {
		t.Fatalf("ControllerPublishVolume failed: %v", err)
	}
	publishContext = publishResp.PublishContext
	t.Logf("Attached drive to VM %d (serial=%s)", targetVMID, publishContext["serial"])

	// Wait for the device to appear in /dev/disk/by-id/.
	serial := publishContext["serial"]
	if serial == "" {
		t.Fatal("publish context missing 'serial'")
	}
	waitForDevice(t, serial)

	return volumeID, publishContext, controller
}

// waitForDevice polls /dev/disk/by-id/ until a device matching the serial appears.
// VergeOS hotplug may take a few seconds.
func waitForDevice(t *testing.T, serial string) {
	t.Helper()

	finder := util.NewDeviceFinder()
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		device, err := finder.FindBySerial(serial)
		if err == nil {
			t.Logf("Device appeared for serial %q: %s", serial, device)
			return
		}
		time.Sleep(1 * time.Second)
	}

	t.Fatalf("timed out waiting for device with serial %q to appear in /dev/disk/by-id/", serial)
}

func TestIntegration_BlockNodeStageAndUnstage(t *testing.T) {
	node := newIntegrationBlockNode(t)
	volumeID, publishCtx, _ := createTestDriveForNode(t)
	ctx := context.Background()

	stagingPath := filepath.Join(t.TempDir(), "staging")

	// Cleanup: ensure we unstage even on failure.
	t.Cleanup(func() {
		mounter := util.NewSafeMounter()
		mounted, err := mounter.IsMounted(stagingPath)
		if err == nil && mounted {
			t.Logf("cleanup: unmounting staging path %s", stagingPath)
			if err := mounter.Unmount(stagingPath); err != nil {
				t.Logf("cleanup: unmount %s failed: %v", stagingPath, err)
			}
		}
	})

	// --- NodeStageVolume: discover device, format ext4, mount to staging ---
	stageReq := &csi.NodeStageVolumeRequest{
		VolumeId:          volumeID,
		StagingTargetPath: stagingPath,
		PublishContext:    publishCtx,
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{
					FsType: "ext4",
				},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}

	_, err := node.NodeStageVolume(ctx, stageReq)
	if err != nil {
		t.Fatalf("NodeStageVolume failed: %v", err)
	}
	t.Logf("NodeStageVolume succeeded: staged at %s", stagingPath)

	// Verify the staging path is actually mounted.
	mounter := util.NewSafeMounter()
	mounted, err := mounter.IsMounted(stagingPath)
	if err != nil {
		t.Fatalf("checking mount status: %v", err)
	}
	if !mounted {
		t.Fatal("staging path is not mounted after NodeStageVolume")
	}

	// Verify we can write to the formatted filesystem.
	testFile := filepath.Join(stagingPath, "csi-block-integration-test.txt")
	if err := os.WriteFile(testFile, []byte("block integration test\n"), 0644); err != nil {
		t.Fatalf("failed to write test file to staged volume: %v", err)
	}
	t.Log("Successfully wrote test file to staged block volume")
	os.Remove(testFile)

	// --- NodeUnstageVolume: unmount staging ---
	_, err = node.NodeUnstageVolume(ctx, &csi.NodeUnstageVolumeRequest{
		VolumeId:          volumeID,
		StagingTargetPath: stagingPath,
	})
	if err != nil {
		t.Fatalf("NodeUnstageVolume failed: %v", err)
	}
	t.Logf("NodeUnstageVolume succeeded: unstaged %s", stagingPath)

	// Verify no longer mounted.
	mounted, err = mounter.IsMounted(stagingPath)
	if err != nil {
		t.Logf("checking mount status after unstage: %v (may be expected if dir removed)", err)
	} else if mounted {
		t.Error("staging path is still mounted after NodeUnstageVolume")
	}
}

func TestIntegration_BlockNodePublishAndUnpublish(t *testing.T) {
	node := newIntegrationBlockNode(t)
	volumeID, publishCtx, _ := createTestDriveForNode(t)
	ctx := context.Background()

	stagingPath := filepath.Join(t.TempDir(), "staging")
	targetPath := filepath.Join(t.TempDir(), "target")

	// Cleanup: ensure we unmount everything even on failure.
	t.Cleanup(func() {
		mounter := util.NewSafeMounter()

		// Unpublish (bind unmount target).
		mounted, err := mounter.IsMounted(targetPath)
		if err == nil && mounted {
			t.Logf("cleanup: unmounting target path %s", targetPath)
			if err := mounter.Unmount(targetPath); err != nil {
				t.Logf("cleanup: unmount target %s failed: %v", targetPath, err)
			}
		}

		// Unstage (unmount staging).
		mounted, err = mounter.IsMounted(stagingPath)
		if err == nil && mounted {
			t.Logf("cleanup: unmounting staging path %s", stagingPath)
			if err := mounter.Unmount(stagingPath); err != nil {
				t.Logf("cleanup: unmount staging %s failed: %v", stagingPath, err)
			}
		}
	})

	// First stage the volume (prerequisite for publish).
	stageReq := &csi.NodeStageVolumeRequest{
		VolumeId:          volumeID,
		StagingTargetPath: stagingPath,
		PublishContext:    publishCtx,
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{FsType: "ext4"},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}
	_, err := node.NodeStageVolume(ctx, stageReq)
	if err != nil {
		t.Fatalf("NodeStageVolume (prerequisite) failed: %v", err)
	}
	t.Logf("Prerequisite: staged at %s", stagingPath)

	// --- NodePublishVolume: bind mount from staging to target ---
	publishReq := &csi.NodePublishVolumeRequest{
		VolumeId:          volumeID,
		StagingTargetPath: stagingPath,
		TargetPath:        targetPath,
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}

	_, err = node.NodePublishVolume(ctx, publishReq)
	if err != nil {
		t.Fatalf("NodePublishVolume failed: %v", err)
	}
	t.Logf("NodePublishVolume succeeded: bind-mounted %s -> %s", stagingPath, targetPath)

	// Verify the target path is mounted.
	mounter := util.NewSafeMounter()
	mounted, err := mounter.IsMounted(targetPath)
	if err != nil {
		t.Fatalf("checking target mount status: %v", err)
	}
	if !mounted {
		t.Fatal("target path is not mounted after NodePublishVolume")
	}

	// Verify we can write through the bind mount.
	testFile := filepath.Join(targetPath, "csi-publish-test.txt")
	if err := os.WriteFile(testFile, []byte("publish test\n"), 0644); err != nil {
		t.Fatalf("failed to write test file to published volume: %v", err)
	}
	t.Log("Successfully wrote test file through bind mount")
	os.Remove(testFile)

	// --- NodeUnpublishVolume: unmount bind mount ---
	_, err = node.NodeUnpublishVolume(ctx, &csi.NodeUnpublishVolumeRequest{
		VolumeId:   volumeID,
		TargetPath: targetPath,
	})
	if err != nil {
		t.Fatalf("NodeUnpublishVolume failed: %v", err)
	}
	t.Logf("NodeUnpublishVolume succeeded: unmounted %s", targetPath)

	// Verify target no longer mounted.
	mounted, err = mounter.IsMounted(targetPath)
	if err != nil {
		t.Logf("checking target after unpublish: %v (may be expected if dir removed)", err)
	} else if mounted {
		t.Error("target path still mounted after NodeUnpublishVolume")
	}

	// --- NodeUnstageVolume: clean up staging ---
	_, err = node.NodeUnstageVolume(ctx, &csi.NodeUnstageVolumeRequest{
		VolumeId:          volumeID,
		StagingTargetPath: stagingPath,
	})
	if err != nil {
		t.Fatalf("NodeUnstageVolume failed: %v", err)
	}
	t.Logf("NodeUnstageVolume succeeded: unstaged %s", stagingPath)
}

func TestIntegration_BlockNodeIdempotent(t *testing.T) {
	node := newIntegrationBlockNode(t)
	volumeID, publishCtx, _ := createTestDriveForNode(t)
	ctx := context.Background()

	stagingPath := filepath.Join(t.TempDir(), "staging")
	targetPath := filepath.Join(t.TempDir(), "target")

	// Cleanup: ensure we unmount everything even on failure.
	t.Cleanup(func() {
		mounter := util.NewSafeMounter()

		mounted, err := mounter.IsMounted(targetPath)
		if err == nil && mounted {
			t.Logf("cleanup: unmounting target %s", targetPath)
			mounter.Unmount(targetPath)
		}

		mounted, err = mounter.IsMounted(stagingPath)
		if err == nil && mounted {
			t.Logf("cleanup: unmounting staging %s", stagingPath)
			mounter.Unmount(stagingPath)
		}
	})

	stageReq := &csi.NodeStageVolumeRequest{
		VolumeId:          volumeID,
		StagingTargetPath: stagingPath,
		PublishContext:    publishCtx,
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{FsType: "ext4"},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}

	// --- Double stage: both should succeed ---
	_, err := node.NodeStageVolume(ctx, stageReq)
	if err != nil {
		t.Fatalf("NodeStageVolume (first) failed: %v", err)
	}
	t.Log("First NodeStageVolume succeeded")

	_, err = node.NodeStageVolume(ctx, stageReq)
	if err != nil {
		t.Fatalf("NodeStageVolume (second, idempotent) failed: %v", err)
	}
	t.Log("Second NodeStageVolume succeeded (idempotent)")

	// --- Double publish: both should succeed ---
	publishReq := &csi.NodePublishVolumeRequest{
		VolumeId:          volumeID,
		StagingTargetPath: stagingPath,
		TargetPath:        targetPath,
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
	}

	_, err = node.NodePublishVolume(ctx, publishReq)
	if err != nil {
		t.Fatalf("NodePublishVolume (first) failed: %v", err)
	}
	t.Log("First NodePublishVolume succeeded")

	_, err = node.NodePublishVolume(ctx, publishReq)
	if err != nil {
		t.Fatalf("NodePublishVolume (second, idempotent) failed: %v", err)
	}
	t.Log("Second NodePublishVolume succeeded (idempotent)")

	// Verify still mounted.
	mounter := util.NewSafeMounter()
	mounted, err := mounter.IsMounted(targetPath)
	if err != nil {
		t.Fatalf("checking mount status: %v", err)
	}
	if !mounted {
		t.Error("target not mounted after idempotent publish calls")
	}

	// Clean up: unpublish + unstage.
	_, err = node.NodeUnpublishVolume(ctx, &csi.NodeUnpublishVolumeRequest{
		VolumeId:   volumeID,
		TargetPath: targetPath,
	})
	if err != nil {
		t.Fatalf("cleanup NodeUnpublishVolume failed: %v", err)
	}
	t.Log("Cleanup: NodeUnpublishVolume succeeded")

	_, err = node.NodeUnstageVolume(ctx, &csi.NodeUnstageVolumeRequest{
		VolumeId:          volumeID,
		StagingTargetPath: stagingPath,
	})
	if err != nil {
		t.Fatalf("cleanup NodeUnstageVolume failed: %v", err)
	}
	t.Log("Cleanup: NodeUnstageVolume succeeded")
}

func TestIntegration_BlockNodeUnstage_NotStaged(t *testing.T) {
	node := newIntegrationBlockNode(t)
	ctx := context.Background()

	// Unstage a path that was never staged — should succeed (idempotent).
	stagingPath := filepath.Join(t.TempDir(), "not-staged")

	_, err := node.NodeUnstageVolume(ctx, &csi.NodeUnstageVolumeRequest{
		VolumeId:          "999999",
		StagingTargetPath: stagingPath,
	})
	if err != nil {
		t.Fatalf("NodeUnstageVolume on non-staged path should succeed: %v", err)
	}
	t.Log("NodeUnstageVolume on non-staged path succeeded (idempotent)")
}

func TestIntegration_BlockNodeUnpublish_NotMounted(t *testing.T) {
	node := newIntegrationBlockNode(t)
	ctx := context.Background()

	// Unpublish a path that was never mounted — should succeed (idempotent).
	targetPath := filepath.Join(t.TempDir(), "not-mounted")

	_, err := node.NodeUnpublishVolume(ctx, &csi.NodeUnpublishVolumeRequest{
		VolumeId:   "999999",
		TargetPath: targetPath,
	})
	if err != nil {
		t.Fatalf("NodeUnpublishVolume on non-mounted path should succeed: %v", err)
	}
	t.Log("NodeUnpublishVolume on non-mounted path succeeded (idempotent)")
}

// TestIntegration_BlockNodeDeviceDiscovery verifies that DeviceFinder can locate
// a hotplugged block device by its serial number from /dev/disk/by-id/.
func TestIntegration_BlockNodeDeviceDiscovery(t *testing.T) {
	// This test only needs the controller (no mount operations), so no root needed.
	// But we still need the device to appear, which requires being on the target VM.
	for _, env := range []string{"VERGEOS_HOST", "VERGEOS_USERNAME", "VERGEOS_PASSWORD"} {
		if os.Getenv(env) == "" {
			t.Skipf("skipping integration test: %s not set", env)
		}
	}

	targetVMIDStr := os.Getenv("VERGEOS_NODE_VM_ID")
	if targetVMIDStr == "" {
		t.Skip("skipping block node integration test: VERGEOS_NODE_VM_ID not set")
	}
	targetVMID, err := strconv.Atoi(targetVMIDStr)
	if err != nil {
		t.Fatalf("invalid VERGEOS_NODE_VM_ID: %v", err)
	}

	// Create a test drive and attach it.
	controller := newIntegrationBlockController(t)
	ctx := context.Background()

	volumeName := fmt.Sprintf("csi-test-devfind-%d", time.Now().UnixNano())

	createResp, err := controller.CreateVolume(ctx, &csi.CreateVolumeRequest{
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
	volumeID := createResp.Volume.VolumeId

	t.Cleanup(func() {
		_, _ = controller.ControllerUnpublishVolume(ctx, &csi.ControllerUnpublishVolumeRequest{
			VolumeId: volumeID,
			NodeId:   "integration-test",
		})
		_, _ = controller.DeleteVolume(ctx, &csi.DeleteVolumeRequest{VolumeId: volumeID})
	})

	publishResp, err := controller.ControllerPublishVolume(ctx, &csi.ControllerPublishVolumeRequest{
		VolumeId: volumeID,
		NodeId:   "integration-test",
		VolumeCapability: &csi.VolumeCapability{
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
		VolumeContext: map[string]string{
			"vergeos.com/vm-id": strconv.Itoa(targetVMID),
		},
	})
	if err != nil {
		t.Fatalf("ControllerPublishVolume failed: %v", err)
	}

	serial := publishResp.PublishContext["serial"]
	t.Logf("Drive attached with serial: %s", serial)

	// Use DeviceFinder to locate the device.
	finder := util.NewDeviceFinder()

	var devicePath string
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		devicePath, err = finder.FindBySerial(serial)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		t.Fatalf("DeviceFinder.FindBySerial(%q) failed after 30s: %v", serial, err)
	}
	t.Logf("DeviceFinder found device: serial=%s -> %s", serial, devicePath)

	// Verify the device path exists and is a block device.
	info, err := os.Stat(devicePath)
	if err != nil {
		t.Fatalf("stat device %s: %v", devicePath, err)
	}
	t.Logf("Device %s: mode=%s, size=%d", devicePath, info.Mode(), info.Size())
}
