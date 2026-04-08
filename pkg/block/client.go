package block

import (
	"context"

	vergeos "github.com/verge-io/govergeos"
)

// VMDriveClient abstracts the govergeos VMDrives API for testing.
type VMDriveClient interface {
	List(ctx context.Context, vmID int) ([]vergeos.VMDrive, error)
	Get(ctx context.Context, driveID int) (*vergeos.VMDrive, error)
	Create(ctx context.Context, vmID int, req *vergeos.VMDriveCreateRequest) (*vergeos.VMDrive, error)
	Update(ctx context.Context, driveID int, req *vergeos.VMDriveUpdateRequest) (*vergeos.VMDrive, error)
	Delete(ctx context.Context, driveID int) error
	HotplugDrive(ctx context.Context, vmID, driveID int) error
	HotUnplugDrive(ctx context.Context, vmID, driveID int) error
}

// VMClient abstracts the govergeos VMs API for testing.
type VMClient interface {
	Get(ctx context.Context, id int) (*vergeos.VM, error)
	List(ctx context.Context, opts ...vergeos.ListOption) ([]vergeos.VM, error)
}
