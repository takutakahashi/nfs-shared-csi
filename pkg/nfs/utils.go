package nfs

import (
	"context"
	"fmt"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	klog.V(4).Infof("GRPC call: %s", info.FullMethod)
	klog.V(5).Infof("GRPC request: %+v", req)

	resp, err := handler(ctx, req)
	if err != nil {
		klog.Errorf("GRPC error: %v", err)
	} else {
		klog.V(5).Infof("GRPC response: %+v", resp)
	}
	return resp, err
}

// validateVolumeCapability checks if the given capability is supported
func validateVolumeCapability(cap *csi.VolumeCapability) error {
	if cap == nil {
		return status.Error(codes.InvalidArgument, "volume capability is nil")
	}

	accessMode := cap.GetAccessMode()
	if accessMode == nil {
		return status.Error(codes.InvalidArgument, "volume capability access mode is nil")
	}

	mode := accessMode.GetMode()
	switch mode {
	case csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER,
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER:
		// Supported
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported access mode: %v", mode)
	}

	accessType := cap.GetAccessType()
	if accessType == nil {
		return status.Error(codes.InvalidArgument, "volume capability access type is nil")
	}

	if _, ok := accessType.(*csi.VolumeCapability_Mount); !ok {
		return status.Error(codes.InvalidArgument, "only mount access type is supported")
	}

	return nil
}

// getVolumeSource extracts server and share from volume context
func getVolumeSource(volumeContext map[string]string) (string, string, error) {
	server := volumeContext[ParamServer]
	if server == "" {
		return "", "", fmt.Errorf("server parameter is required")
	}

	share := volumeContext[ParamShare]
	if share == "" {
		return "", "", fmt.Errorf("share parameter is required")
	}

	// Ensure share starts with /
	if !strings.HasPrefix(share, "/") {
		share = "/" + share
	}

	return server, share, nil
}
