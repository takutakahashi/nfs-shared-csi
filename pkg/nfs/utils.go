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

// getVolumeSource extracts server, share and subPath from volume context
// subPath can be specified via:
// 1. volumeContext["subPath"] (from PV volumeAttributes)
// 2. PVC annotation "nfs.csi.takutakahashi.dev/subPath" (passed via csi.storage.k8s.io/pvc/annotations)
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

	// Get subPath from volumeContext or PVC annotation
	subPath := getSubPath(volumeContext)
	if subPath != "" {
		// Combine share with subPath
		share = strings.TrimSuffix(share, "/") + "/" + strings.TrimPrefix(subPath, "/")
	}

	return server, share, nil
}

// getSubPath extracts subPath from volume context
// Priority: 1. volumeContext["subPath"], 2. PVC annotation
func getSubPath(volumeContext map[string]string) string {
	// First, check direct subPath parameter
	if subPath := volumeContext[ParamSubPath]; subPath != "" {
		return subPath
	}

	// Check PVC annotation (passed by CSI external-provisioner)
	// The annotation key format is: csi.storage.k8s.io/pvc/annotations
	// Value is JSON-encoded annotations map
	if annotations := volumeContext["csi.storage.k8s.io/pvc/annotations"]; annotations != "" {
		// Parse JSON annotations and extract subPath
		subPath := parseAnnotationSubPath(annotations)
		if subPath != "" {
			return subPath
		}
	}

	return ""
}

// parseAnnotationSubPath extracts subPath from JSON-encoded PVC annotations
func parseAnnotationSubPath(annotationsJSON string) string {
	// Simple parsing for the annotation key
	// Format: {"nfs.csi.takutakahashi.dev/subPath":"value",...}
	key := fmt.Sprintf(`"%s":"`, AnnotationSubPath)
	idx := strings.Index(annotationsJSON, key)
	if idx == -1 {
		return ""
	}

	start := idx + len(key)
	end := strings.Index(annotationsJSON[start:], `"`)
	if end == -1 {
		return ""
	}

	return annotationsJSON[start : start+end]
}
