package block

import (
	"crypto/sha256"
	"fmt"
	"strconv"
)

// encodeVolumeID converts a VergeOS drive ID to a CSI volume ID string.
func encodeVolumeID(driveID int) string {
	return strconv.Itoa(driveID)
}

// decodeVolumeID parses a CSI volume ID string back to a VergeOS drive ID.
func decodeVolumeID(csiVolumeID string) (int, error) {
	id, err := strconv.Atoi(csiVolumeID)
	if err != nil {
		return 0, fmt.Errorf("invalid block volume ID %q: %w", csiVolumeID, err)
	}
	return id, nil
}

// deriveSerial creates a deterministic, short serial number from the CSI volume
// name. This serial is set on the VM drive so the node can discover the block
// device via /dev/disk/by-id/ after hotplug.
//
// Max 20 characters (virtio serial limit). Uses first 16 hex chars of SHA-256.
func deriveSerial(csiVolumeName string) string {
	hash := sha256.Sum256([]byte(csiVolumeName))
	return fmt.Sprintf("csi-%x", hash[:8]) // "csi-" + 16 hex chars = 20 chars
}
