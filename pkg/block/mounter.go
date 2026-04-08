package block

// BlockMounter abstracts mount and filesystem operations for testing.
type BlockMounter interface {
	// IsMounted returns true if the given path is a mount point.
	IsMounted(path string) (bool, error)

	// FormatAndMount formats a device with the given filesystem type (if needed)
	// and mounts it to the target path.
	FormatAndMount(source, target, fsType string, options []string) error

	// Mount mounts source to target with the given filesystem type and options.
	Mount(source, target, fsType string, options []string) error

	// Unmount unmounts the given target path.
	Unmount(target string) error
}

// DeviceDiscovery abstracts block device discovery for testing.
type DeviceDiscovery interface {
	// FindBySerial finds a block device by its serial number.
	FindBySerial(serial string) (string, error)
}
