package util

import (
	"fmt"
	"os"

	"k8s.io/klog/v2"
	"k8s.io/mount-utils"
	utilexec "k8s.io/utils/exec"
)

// Mounter wraps k8s.io/mount-utils for CSI mount operations.
type Mounter struct {
	mount.Interface
}

// NewMounter creates a new Mounter using the default OS mounter.
func NewMounter() *Mounter {
	return &Mounter{
		Interface: mount.New(""),
	}
}

// EnsureDirectory creates a directory (and parents) if it doesn't exist.
func EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0750)
}

// CleanupMountPoint removes a mount point directory if it exists.
// It does NOT unmount — call Unmount first.
func CleanupMountPoint(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing mount point %q: %w", path, err)
	}
	return nil
}

// IsMounted checks if the given path is a mount point.
func (m *Mounter) IsMounted(path string) (bool, error) {
	notMnt, err := m.IsLikelyNotMountPoint(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !notMnt, nil
}

// MountNFS mounts an NFS share to a target path.
func (m *Mounter) MountNFS(source, target string, options []string) error {
	if err := EnsureDirectory(target); err != nil {
		return fmt.Errorf("creating mount target %q: %w", target, err)
	}

	klog.Infof("NFS mount: %s -> %s (options: %v)", source, target, options)
	return m.Mount(source, target, "nfs", options)
}

// Unmount unmounts the given path.
func (m *Mounter) Unmount(target string) error {
	mounted, err := m.IsMounted(target)
	if err != nil {
		return fmt.Errorf("checking mount status of %q: %w", target, err)
	}
	if !mounted {
		klog.Infof("Path %q is not mounted, nothing to unmount", target)
		return nil
	}

	klog.Infof("Unmounting %s", target)
	return m.Interface.Unmount(target)
}

// SafeMounter extends Mounter with format-and-mount capability for block devices.
type SafeMounter struct {
	*Mounter
	safe *mount.SafeFormatAndMount
}

// NewSafeMounter creates a SafeMounter that can format and mount block devices.
func NewSafeMounter() *SafeMounter {
	mounter := NewMounter()
	return &SafeMounter{
		Mounter: mounter,
		safe: &mount.SafeFormatAndMount{
			Interface: mounter.Interface,
			Exec:      utilexec.New(),
		},
	}
}

// FormatAndMount formats the device with fsType (if not already formatted)
// and mounts it to the target path.
func (m *SafeMounter) FormatAndMount(source, target, fsType string, options []string) error {
	klog.Infof("FormatAndMount: device=%s, target=%s, fsType=%s", source, target, fsType)
	return m.safe.FormatAndMount(source, target, fsType, options)
}

// Mount performs a regular mount (used for bind mounts).
func (m *SafeMounter) Mount(source, target, fsType string, options []string) error {
	klog.Infof("Mount: source=%s, target=%s, fsType=%s, options=%v", source, target, fsType, options)
	return m.Mounter.Interface.Mount(source, target, fsType, options)
}
