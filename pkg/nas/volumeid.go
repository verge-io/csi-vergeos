package nas

import (
	"fmt"
	"strconv"
	"strings"
)

// encodeVolumeID creates a composite CSI volume ID from the NAS service ID,
// VergeOS volume ID (SHA1), and NFS share ID (SHA1).
func encodeVolumeID(nasServiceID int, volumeID, nfsShareID string) string {
	return fmt.Sprintf("%d:%s:%s", nasServiceID, volumeID, nfsShareID)
}

// decodeVolumeID parses a composite CSI volume ID back into its components.
func decodeVolumeID(csiVolumeID string) (nasServiceID int, volumeID, nfsShareID string, err error) {
	parts := strings.SplitN(csiVolumeID, ":", 3)
	if len(parts) != 3 {
		return 0, "", "", fmt.Errorf("invalid NAS volume ID %q: expected format <nasServiceID>:<volumeID>:<nfsShareID>", csiVolumeID)
	}

	nasServiceID, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid NAS service ID in volume ID %q: %w", csiVolumeID, err)
	}

	return nasServiceID, parts[1], parts[2], nil
}
