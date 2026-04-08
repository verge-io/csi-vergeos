package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/klog/v2"
)

const defaultDiskByIDPath = "/dev/disk/by-id/"

// DeviceFinder discovers block devices by serial number.
type DeviceFinder struct {
	diskByIDPath string
}

// NewDeviceFinder creates a DeviceFinder using the default /dev/disk/by-id/ path.
func NewDeviceFinder() *DeviceFinder {
	return &DeviceFinder{diskByIDPath: defaultDiskByIDPath}
}

// FindBySerial scans /dev/disk/by-id/ for a device matching the given serial.
// Returns the resolved device path (e.g., /dev/sdb).
func (f *DeviceFinder) FindBySerial(serial string) (string, error) {
	entries, err := os.ReadDir(f.diskByIDPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", f.diskByIDPath, err)
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip partition entries (e.g., ...-part1).
		if strings.Contains(name, "-part") {
			continue
		}

		// Check if the entry name contains the serial.
		if !strings.Contains(name, serial) {
			continue
		}

		// Resolve the symlink to get the real device path.
		linkPath := filepath.Join(f.diskByIDPath, name)
		resolved, err := filepath.EvalSymlinks(linkPath)
		if err != nil {
			klog.Warningf("Failed to resolve symlink %s: %v", linkPath, err)
			continue
		}

		klog.Infof("Found device for serial %q: %s -> %s", serial, linkPath, resolved)
		return resolved, nil
	}

	return "", fmt.Errorf("no block device found with serial %q in %s", serial, f.diskByIDPath)
}
