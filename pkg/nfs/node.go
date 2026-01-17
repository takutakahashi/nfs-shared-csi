package nfs

import (
	"context"
	"fmt"
	"os"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"k8s.io/mount-utils"
)

// NodePublishVolume mounts the NFS share at the target path
func (d *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()
	volumeContext := req.GetVolumeContext()

	klog.V(2).Infof("NodePublishVolume: volumeID=%s, targetPath=%s, volumeContext=%+v", volumeID, targetPath, volumeContext)

	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path is required")
	}

	cap := req.GetVolumeCapability()
	if err := validateVolumeCapability(cap); err != nil {
		return nil, err
	}

	// Log subPath extraction process
	if subPath := getSubPath(volumeContext); subPath != "" {
		klog.V(2).Infof("NodePublishVolume: subPath found in volumeContext: %s", subPath)
	} else {
		klog.V(2).Infof("NodePublishVolume: No subPath found in volumeContext")
	}

	server, share, err := getVolumeSource(volumeContext)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get volume source: %v", err)
	}

	klog.V(2).Infof("NodePublishVolume: After processing - server=%s, share=%s (share may include subPath)", server, share)

	source := fmt.Sprintf("%s:%s", server, share)
	klog.V(4).Infof("Mounting NFS: source=%s, target=%s", source, targetPath)

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetPath, 0750); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create target path %s: %v", targetPath, err)
	}

	// Check if already mounted
	notMnt, err := d.mounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			notMnt = true
		} else {
			return nil, status.Errorf(codes.Internal, "failed to check mount point: %v", err)
		}
	}

	if !notMnt {
		klog.V(2).Infof("Target path %s is already mounted", targetPath)
		return &csi.NodePublishVolumeResponse{}, nil
	}

	// Prepare mount options with default NFS options
	// nolock: disable NFS locking (avoids rpc.statd requirement in containers)
	mountOptions := []string{"nolock"}

	// Get mount options from volume capability
	if mountCap := cap.GetMount(); mountCap != nil {
		mountOptions = append(mountOptions, mountCap.GetMountFlags()...)
	}

	// Handle read-only mount
	if req.GetReadonly() {
		mountOptions = append(mountOptions, "ro")
	}

	klog.V(4).Infof("Mount options: %v", mountOptions)

	// Mount NFS
	if err := d.mounter.Mount(source, targetPath, "nfs", mountOptions); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mount NFS %s at %s: %v", source, targetPath, err)
	}

	klog.V(2).Infof("Successfully mounted NFS %s at %s", source, targetPath)
	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unmounts the NFS share from the target path
func (d *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()

	klog.V(2).Infof("NodeUnpublishVolume: volumeID=%s, targetPath=%s", volumeID, targetPath)

	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID is required")
	}
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "target path is required")
	}

	// Check if mounted
	notMnt, err := d.mounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			klog.V(4).Infof("Target path %s does not exist, nothing to unmount", targetPath)
			return &csi.NodeUnpublishVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to check mount point: %v", err)
	}

	if notMnt {
		klog.V(4).Infof("Target path %s is not mounted", targetPath)
		// Clean up directory
		if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
			klog.Warningf("Failed to remove target path %s: %v", targetPath, err)
		}
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	// Unmount
	if err := mount.CleanupMountPoint(targetPath, d.mounter, true); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmount %s: %v", targetPath, err)
	}

	klog.V(2).Infof("Successfully unmounted %s", targetPath)
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeGetCapabilities returns the capabilities of the node service
func (d *Driver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	klog.V(4).Infof("NodeGetCapabilities called")

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{},
	}, nil
}

// NodeGetInfo returns information about the node
func (d *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	klog.V(4).Infof("NodeGetInfo called")

	return &csi.NodeGetInfoResponse{
		NodeId: d.nodeID,
	}, nil
}

// NodeStageVolume is not implemented for NFS
func (d *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeStageVolume is not implemented")
}

// NodeUnstageVolume is not implemented for NFS
func (d *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeUnstageVolume is not implemented")
}

// NodeGetVolumeStats is not implemented
func (d *Driver) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeGetVolumeStats is not implemented")
}

// NodeExpandVolume is not implemented
func (d *Driver) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "NodeExpandVolume is not implemented")
}
