package driver

// DriverVersion is the current version of the CSI driver.
// This is a var so it can be overridden at build time via ldflags:
//
//	go build -ldflags="-X github.com/verge-io/csi-vergeos/pkg/driver.DriverVersion=v1.0.0"
var DriverVersion = "0.1.0"

const (
	// NASDriverName is the CSI driver name for NAS-backed volumes.
	NASDriverName = "csi-nas.verge.io"

	// BlockDriverName is the CSI driver name for block-backed volumes.
	BlockDriverName = "csi-block.verge.io"

	// VolumeNamePrefix is prepended to CSI volume names when creating
	// VergeOS resources, ensuring CSI-managed resources are identifiable.
	VolumeNamePrefix = "csi-"
)

// DriverMode selects which gRPC services the binary runs.
type DriverMode string

const (
	ControllerMode DriverMode = "controller"
	NodeMode       DriverMode = "node"
)

// DriverType selects which CSI driver name to register.
type DriverType string

const (
	NASDriver   DriverType = "nas"
	BlockDriver DriverType = "block"
)

// DriverNameForType returns the CSI driver name for the given driver type.
func DriverNameForType(dt DriverType) string {
	switch dt {
	case NASDriver:
		return NASDriverName
	case BlockDriver:
		return BlockDriverName
	default:
		return ""
	}
}
