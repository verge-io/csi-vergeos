package block

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- Mocks ---

type mockBlockMounter struct {
	mountedPaths map[string]bool
	formatErr    error
	mountErr     error
	unmountErr   error
}

func newMockBlockMounter() *mockBlockMounter {
	return &mockBlockMounter{mountedPaths: make(map[string]bool)}
}

func (m *mockBlockMounter) IsMounted(path string) (bool, error) {
	return m.mountedPaths[path], nil
}

func (m *mockBlockMounter) FormatAndMount(source, target, fsType string, options []string) error {
	if m.formatErr != nil {
		return m.formatErr
	}
	m.mountedPaths[target] = true
	return nil
}

func (m *mockBlockMounter) Mount(source, target, fsType string, options []string) error {
	if m.mountErr != nil {
		return m.mountErr
	}
	m.mountedPaths[target] = true
	return nil
}

func (m *mockBlockMounter) Unmount(target string) error {
	if m.unmountErr != nil {
		return m.unmountErr
	}
	delete(m.mountedPaths, target)
	return nil
}

type mockDeviceDiscovery struct {
	devicePath string
	findErr    error
}

func (m *mockDeviceDiscovery) FindBySerial(serial string) (string, error) {
	if m.findErr != nil {
		return "", m.findErr
	}
	return m.devicePath, nil
}

// --- Tests ---

func newTestBlockNode() (*BlockNode, *mockBlockMounter, *mockDeviceDiscovery) {
	mounter := newMockBlockMounter()
	discovery := &mockDeviceDiscovery{devicePath: "/dev/sdb"}
	node := &BlockNode{
		mounter:   mounter,
		discovery: discovery,
	}
	return node, mounter, discovery
}

// --- NodeStageVolume Tests ---

func TestNodeStageVolume_Success(t *testing.T) {
	node, mounter, _ := newTestBlockNode()
	stagingPath := filepath.Join(t.TempDir(), "staging", "pvc-123")

	req := &csi.NodeStageVolumeRequest{
		VolumeId:          "100",
		StagingTargetPath: stagingPath,
		PublishContext: map[string]string{
			"serial": "csi-abc123",
		},
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

	_, err := node.NodeStageVolume(context.Background(), req)
	if err != nil {
		t.Fatalf("NodeStageVolume failed: %v", err)
	}

	if !mounter.mountedPaths[stagingPath] {
		t.Error("expected staging path to be mounted")
	}
}

func TestNodeStageVolume_AlreadyStaged(t *testing.T) {
	node, mounter, _ := newTestBlockNode()
	stagingPath := "/var/lib/kubelet/plugins/csi-block.verge.io/staging/pvc-123"
	mounter.mountedPaths[stagingPath] = true

	req := &csi.NodeStageVolumeRequest{
		VolumeId:          "100",
		StagingTargetPath: stagingPath,
		PublishContext:    map[string]string{"serial": "csi-abc123"},
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{FsType: "ext4"}},
			AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER},
		},
	}

	_, err := node.NodeStageVolume(context.Background(), req)
	if err != nil {
		t.Fatalf("NodeStageVolume (idempotent) should succeed: %v", err)
	}
}

func TestNodeStageVolume_DeviceNotFound(t *testing.T) {
	node, _, discovery := newTestBlockNode()
	discovery.findErr = fmt.Errorf("no device found")

	req := &csi.NodeStageVolumeRequest{
		VolumeId:          "100",
		StagingTargetPath: "/staging",
		PublishContext:    map[string]string{"serial": "csi-bad"},
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{FsType: "ext4"}},
			AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER},
		},
	}

	_, err := node.NodeStageVolume(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when device not found")
	}
}

func TestNodeStageVolume_MissingSerial(t *testing.T) {
	node, _, _ := newTestBlockNode()

	req := &csi.NodeStageVolumeRequest{
		VolumeId:          "100",
		StagingTargetPath: "/staging",
		PublishContext:    map[string]string{},
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{FsType: "ext4"}},
			AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER},
		},
	}

	_, err := node.NodeStageVolume(context.Background(), req)
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

// --- NodeUnstageVolume Tests ---

func TestNodeUnstageVolume_Success(t *testing.T) {
	node, mounter, _ := newTestBlockNode()
	stagingPath := "/var/lib/kubelet/plugins/csi-block.verge.io/staging/pvc-123"
	mounter.mountedPaths[stagingPath] = true

	_, err := node.NodeUnstageVolume(context.Background(), &csi.NodeUnstageVolumeRequest{
		VolumeId:          "100",
		StagingTargetPath: stagingPath,
	})
	if err != nil {
		t.Fatalf("NodeUnstageVolume failed: %v", err)
	}
	if mounter.mountedPaths[stagingPath] {
		t.Error("expected staging path to be unmounted")
	}
}

func TestNodeUnstageVolume_NotStaged(t *testing.T) {
	node, _, _ := newTestBlockNode()

	_, err := node.NodeUnstageVolume(context.Background(), &csi.NodeUnstageVolumeRequest{
		VolumeId:          "100",
		StagingTargetPath: "/not/staged",
	})
	if err != nil {
		t.Fatalf("NodeUnstageVolume (idempotent) should succeed: %v", err)
	}
}

// --- NodePublishVolume Tests ---

func TestNodePublishVolume_Success(t *testing.T) {
	node, mounter, _ := newTestBlockNode()
	stagingPath := "/staging/pvc-123"
	mounter.mountedPaths[stagingPath] = true
	targetPath := filepath.Join(t.TempDir(), "pods", "uid", "volumes", "csi", "pvc-123", "mount")

	req := &csi.NodePublishVolumeRequest{
		VolumeId:          "100",
		StagingTargetPath: stagingPath,
		TargetPath:        targetPath,
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}},
			AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER},
		},
	}

	_, err := node.NodePublishVolume(context.Background(), req)
	if err != nil {
		t.Fatalf("NodePublishVolume failed: %v", err)
	}
	if !mounter.mountedPaths[targetPath] {
		t.Error("expected target path to be mounted")
	}
}

func TestNodePublishVolume_AlreadyMounted(t *testing.T) {
	node, mounter, _ := newTestBlockNode()
	targetPath := "/var/lib/kubelet/pods/uid/volumes/csi/pvc-123/mount"
	mounter.mountedPaths[targetPath] = true

	req := &csi.NodePublishVolumeRequest{
		VolumeId:          "100",
		StagingTargetPath: "/staging",
		TargetPath:        targetPath,
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{}},
			AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER},
		},
	}

	_, err := node.NodePublishVolume(context.Background(), req)
	if err != nil {
		t.Fatalf("NodePublishVolume (idempotent) should succeed: %v", err)
	}
}

// --- NodeUnpublishVolume Tests ---

func TestNodeUnpublishVolume_Success(t *testing.T) {
	node, mounter, _ := newTestBlockNode()
	targetPath := "/var/lib/kubelet/pods/uid/mount"
	mounter.mountedPaths[targetPath] = true

	_, err := node.NodeUnpublishVolume(context.Background(), &csi.NodeUnpublishVolumeRequest{
		VolumeId:   "100",
		TargetPath: targetPath,
	})
	if err != nil {
		t.Fatalf("NodeUnpublishVolume failed: %v", err)
	}
	if mounter.mountedPaths[targetPath] {
		t.Error("expected target to be unmounted")
	}
}

func TestNodeUnpublishVolume_NotMounted(t *testing.T) {
	node, _, _ := newTestBlockNode()

	_, err := node.NodeUnpublishVolume(context.Background(), &csi.NodeUnpublishVolumeRequest{
		VolumeId:   "100",
		TargetPath: "/not/mounted",
	})
	if err != nil {
		t.Fatalf("NodeUnpublishVolume (idempotent) should succeed: %v", err)
	}
}

// --- Capabilities ---

func TestNodeGetCapabilities_Block(t *testing.T) {
	node, _, _ := newTestBlockNode()
	caps := node.NodeGetCapabilities()

	hasStage := false
	for _, cap := range caps {
		if rpc := cap.GetRpc(); rpc != nil {
			if rpc.Type == csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME {
				hasStage = true
			}
		}
	}
	if !hasStage {
		t.Error("block node should advertise STAGE_UNSTAGE_VOLUME")
	}
}
