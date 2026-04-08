package nas

// NFSMounter abstracts NFS mount operations for testing.
type NFSMounter interface {
	// IsMounted returns true if the given path is currently a mount point.
	IsMounted(path string) (bool, error)

	// MountNFS mounts an NFS share (source) to the target path.
	MountNFS(source, target string, options []string) error

	// Unmount unmounts the given target path.
	Unmount(target string) error
}
