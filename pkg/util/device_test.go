package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindDeviceBySerial_Found(t *testing.T) {
	// Create a fake /dev/disk/by-id/ directory structure.
	tmpDir := t.TempDir()
	fakeByID := filepath.Join(tmpDir, "disk", "by-id")
	if err := os.MkdirAll(fakeByID, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a fake device file that the symlink points to.
	fakeDev := filepath.Join(tmpDir, "sdb")
	if err := os.WriteFile(fakeDev, nil, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a fake symlink: scsi-0QEMU_QEMU_HARDDISK_csi-abc123 -> ../../sdb
	linkName := filepath.Join(fakeByID, "scsi-0QEMU_QEMU_HARDDISK_csi-abc123")
	if err := os.Symlink("../../sdb", linkName); err != nil {
		t.Fatal(err)
	}

	finder := &DeviceFinder{diskByIDPath: fakeByID}
	device, err := finder.FindBySerial("csi-abc123")
	if err != nil {
		t.Fatalf("FindBySerial failed: %v", err)
	}
	if device == "" {
		t.Error("expected a device path")
	}
}

func TestFindDeviceBySerial_NotFound(t *testing.T) {
	fakeByID := filepath.Join(t.TempDir(), "by-id")
	if err := os.MkdirAll(fakeByID, 0755); err != nil {
		t.Fatal(err)
	}

	finder := &DeviceFinder{diskByIDPath: fakeByID}
	_, err := finder.FindBySerial("nonexistent-serial")
	if err == nil {
		t.Error("expected error when device not found")
	}
}

func TestFindDeviceBySerial_MultipleMatches(t *testing.T) {
	tmpDir := t.TempDir()
	fakeByID := filepath.Join(tmpDir, "disk", "by-id")
	if err := os.MkdirAll(fakeByID, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a fake device file.
	fakeDev := filepath.Join(tmpDir, "sdb")
	if err := os.WriteFile(fakeDev, nil, 0644); err != nil {
		t.Fatal(err)
	}

	// Two entries with the same serial (e.g., scsi- and virtio- prefixes).
	os.Symlink("../../sdb", filepath.Join(fakeByID, "scsi-0QEMU_csi-abc123"))
	os.Symlink("../../sdb", filepath.Join(fakeByID, "virtio-csi-abc123"))

	finder := &DeviceFinder{diskByIDPath: fakeByID}
	device, err := finder.FindBySerial("csi-abc123")
	if err != nil {
		t.Fatalf("FindBySerial with multiple entries failed: %v", err)
	}
	if device == "" {
		t.Error("expected a device path")
	}
}

func TestFindDeviceBySerial_SkipsPartitions(t *testing.T) {
	tmpDir := t.TempDir()
	fakeByID := filepath.Join(tmpDir, "disk", "by-id")
	if err := os.MkdirAll(fakeByID, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a fake device file.
	fakeDev := filepath.Join(tmpDir, "sdb")
	if err := os.WriteFile(fakeDev, nil, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a partition entry (should be skipped) and a whole-disk entry.
	os.Symlink("../../sdb1", filepath.Join(fakeByID, "scsi-0QEMU_csi-abc123-part1"))
	os.Symlink("../../sdb", filepath.Join(fakeByID, "scsi-0QEMU_csi-abc123"))

	finder := &DeviceFinder{diskByIDPath: fakeByID}
	device, err := finder.FindBySerial("csi-abc123")
	if err != nil {
		t.Fatalf("FindBySerial failed: %v", err)
	}
	if device == "" {
		t.Error("expected a device path")
	}
}
