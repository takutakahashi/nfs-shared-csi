package nfs

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// ControllerGetCapabilities returns the capabilities of the controller service
func (d *Driver) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	klog.V(4).Infof("ControllerGetCapabilities called")

	// Support dynamic provisioning
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
					},
				},
			},
		},
	}, nil
}

// ValidateVolumeCapabilities validates the volume capabilities
func (d *Driver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	volumeID := req.GetVolumeId()
	capabilities := req.GetVolumeCapabilities()

	klog.V(4).Infof("ValidateVolumeCapabilities: volumeID=%s", volumeID)

	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}
	if len(capabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume capabilities are required")
	}

	// Validate each capability
	for _, cap := range capabilities {
		if err := validateVolumeCapability(cap); err != nil {
			return &csi.ValidateVolumeCapabilitiesResponse{
				Message: err.Error(),
			}, nil
		}
	}

	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: capabilities,
		},
	}, nil
}

// CreateVolume creates a volume for dynamic provisioning
// Note: This does not create any directories on the NFS server.
// The NFS share must already exist and be properly configured.
func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	volumeName := req.GetName()
	if volumeName == "" {
		return nil, status.Error(codes.InvalidArgument, "volume name is required")
	}

	capabilities := req.GetVolumeCapabilities()
	if len(capabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume capabilities are required")
	}

	// Validate capabilities
	for _, cap := range capabilities {
		if err := validateVolumeCapability(cap); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	// Get NFS server and share from parameters
	parameters := req.GetParameters()
	server := parameters[ParamServer]
	share := parameters[ParamShare]

	// Get subPath from parameters (StorageClass) or PVC annotations
	// Priority: 1. StorageClass parameters, 2. PVC annotation
	subPath := parameters[ParamSubPath]
	if subPath == "" {
		// Try to get from PVC annotations (requires external-provisioner with --extra-create-metadata)
		if annotations := parameters["csi.storage.k8s.io/pvc/annotations"]; annotations != "" {
			subPath = parseAnnotationSubPath(annotations)
			if subPath != "" {
				klog.V(2).Infof("CreateVolume: subPath from PVC annotation: %s", subPath)
			}
		}
	}

	if server == "" {
		return nil, status.Error(codes.InvalidArgument, "server parameter is required")
	}
	if share == "" {
		return nil, status.Error(codes.InvalidArgument, "share parameter is required")
	}

	// Validate subPath if provided
	if subPath != "" {
		if err := validateSubPath(subPath); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid subPath: %v", err)
		}
	}

	klog.V(2).Infof("CreateVolume: name=%s, server=%s, share=%s, subPath=%s", volumeName, server, share, subPath)

	// Generate volume ID
	volumeID := volumeName

	// Build volume context
	volumeContext := map[string]string{
		ParamServer: server,
		ParamShare:  share,
	}
	if subPath != "" {
		volumeContext[ParamSubPath] = subPath
	}

	// Note: We do not create any directories on the NFS server.
	// The NFS share must already exist and be accessible.

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      volumeID,
			VolumeContext: volumeContext,
		},
	}, nil
}

// DeleteVolume deletes a volume
// Note: This does not delete any data on the NFS server.
// The NFS share and its contents remain unchanged.
func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}

	klog.V(2).Infof("DeleteVolume: volumeID=%s", volumeID)

	// Note: We do not delete any directories or data on the NFS server.
	// The NFS share and its contents are managed externally.

	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerPublishVolume is not implemented
func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerPublishVolume is not implemented")
}

// ControllerUnpublishVolume is not implemented
func (d *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerUnpublishVolume is not implemented")
}

// GetCapacity is not implemented
func (d *Driver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetCapacity is not implemented")
}

// ListVolumes is not implemented
func (d *Driver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListVolumes is not implemented")
}

// CreateSnapshot is not implemented
func (d *Driver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CreateSnapshot is not implemented")
}

// DeleteSnapshot is not implemented
func (d *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "DeleteSnapshot is not implemented")
}

// ListSnapshots is not implemented
func (d *Driver) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListSnapshots is not implemented")
}

// ControllerExpandVolume is not implemented
func (d *Driver) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ControllerExpandVolume is not implemented")
}
