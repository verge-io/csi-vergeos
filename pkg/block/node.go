package block

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"

	"github.com/verge-io/csi-vergeos/pkg/util"
)

// BlockNode implements driver.NodeBackend for block volumes.
type BlockNode struct {
	mounter   BlockMounter
	discovery DeviceDiscovery
}

// NewBlockNode creates a block node backend.
func NewBlockNode() *BlockNode {
	return &BlockNode{
		mounter:   util.NewSafeMounter(),
		discovery: util.NewDeviceFinder(),
	}
}

func (n *BlockNode) NodeStageVolume(_ context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "staging target path is required")
	}

	stagingPath := req.StagingTargetPath

	// Idempotency: check if already staged.
	mounted, err := n.mounter.IsMounted(stagingPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "checking mount status: %v", err)
	}
	if mounted {
		klog.Infof("Volume %s already staged at %s (idempotent)", req.VolumeId, stagingPath)
		return &csi.NodeStageVolumeResponse{}, nil
	}

	// Get the drive serial from publish context (set by ControllerPublishVolume).
	serial := req.PublishContext["serial"]
	if serial == "" {
		return nil, status.Error(codes.InvalidArgument, "publish context missing 'serial'")
	}

	// Discover the block device by serial.
	devicePath, err := n.discovery.FindBySerial(serial)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "block device with serial %q not found: %v", serial, err)
	}

	// Determine filesystem type (default ext4).
	fsType := "ext4"
	if mountCap := req.VolumeCapability.GetMount(); mountCap != nil && mountCap.FsType != "" {
		fsType = mountCap.FsType
	}

	// Create staging directory.
	if err := util.EnsureDirectory(stagingPath); err != nil {
		return nil, status.Errorf(codes.Internal, "creating staging directory: %v", err)
	}

	// Format (if needed) and mount the device to the staging path.
	klog.Infof("Staging volume %s: device=%s, fs=%s, target=%s", req.VolumeId, devicePath, fsType, stagingPath)
	if err := n.mounter.FormatAndMount(devicePath, stagingPath, fsType, nil); err != nil {
		return nil, status.Errorf(codes.Internal, "format and mount failed: %v", err)
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

func (n *BlockNode) NodeUnstageVolume(_ context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "staging target path is required")
	}

	stagingPath := req.StagingTargetPath

	// Idempotency: check if not staged.
	mounted, err := n.mounter.IsMounted(stagingPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "checking mount status: %v", err)
	}
	if !mounted {
		klog.Infof("Volume %s not staged at %s (idempotent)", req.VolumeId, stagingPath)
		return &csi.NodeUnstageVolumeResponse{}, nil
	}

	klog.Infof("Unstaging volume %s from %s", req.VolumeId, stagingPath)
	if err := n.mounter.Unmount(stagingPath); err != nil {
		return nil, status.Errorf(codes.Internal, "unmount staging failed: %v", err)
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (n *BlockNode) NodePublishVolume(_ context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}
	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path is required")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "staging target path is required")
	}

	targetPath := req.TargetPath

	// Idempotency: check if already mounted.
	mounted, err := n.mounter.IsMounted(targetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "checking mount status: %v", err)
	}
	if mounted {
		klog.Infof("Volume %s already published at %s (idempotent)", req.VolumeId, targetPath)
		return &csi.NodePublishVolumeResponse{}, nil
	}

	// Create target directory.
	if err := util.EnsureDirectory(targetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "creating target directory: %v", err)
	}

	// Bind-mount from staging to target.
	options := []string{"bind"}
	if req.Readonly {
		options = append(options, "ro")
	}

	klog.Infof("Publishing volume %s: bind mount %s -> %s", req.VolumeId, req.StagingTargetPath, targetPath)
	if err := n.mounter.Mount(req.StagingTargetPath, targetPath, "", options); err != nil {
		return nil, status.Errorf(codes.Internal, "bind mount failed: %v", err)
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (n *BlockNode) NodeUnpublishVolume(_ context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}
	if req.TargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path is required")
	}

	targetPath := req.TargetPath

	// Idempotency: check if not mounted.
	mounted, err := n.mounter.IsMounted(targetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "checking mount status: %v", err)
	}
	if !mounted {
		klog.Infof("Volume %s not published at %s (idempotent)", req.VolumeId, targetPath)
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	klog.Infof("Unpublishing volume %s from %s", req.VolumeId, targetPath)
	if err := n.mounter.Unmount(targetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "unmount failed: %v", err)
	}

	if err := util.CleanupMountPoint(targetPath); err != nil {
		klog.Warningf("Failed to clean up mount point %s: %v", targetPath, err)
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (n *BlockNode) NodeGetCapabilities() []*csi.NodeServiceCapability {
	return []*csi.NodeServiceCapability{
		{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
				},
			},
		},
	}
}
