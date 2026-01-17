package nfs

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
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

const (
	// Maximum allowed length for subPath to prevent potential issues
	maxSubPathLength = 4096
)

// validateSubPath validates that the subPath is safe and doesn't contain path traversal attacks
func validateSubPath(subPath string) error {
	if subPath == "" {
		return nil
	}

	// Check length
	if len(subPath) > maxSubPathLength {
		return fmt.Errorf("subPath exceeds maximum length of %d characters", maxSubPathLength)
	}

	// Clean the path to resolve any .. or . components
	cleaned := filepath.Clean(subPath)

	// Remove leading slash for comparison
	originalNoLeadingSlash := strings.TrimPrefix(subPath, "/")
	cleanedNoLeadingSlash := strings.TrimPrefix(cleaned, "/")

	// Check if the cleaned path contains path traversal attempts
	// After cleaning, if the path starts with .. or contains ../, it's attempting traversal
	if strings.HasPrefix(cleanedNoLeadingSlash, "..") || strings.Contains(cleanedNoLeadingSlash, "/..") {
		return fmt.Errorf("subPath contains path traversal attempt: %s", subPath)
	}

	// Check if cleaning changed the path significantly (excluding leading/trailing slashes)
	// This catches attempts to use . or .. in the path
	originalNormalized := strings.Trim(originalNoLeadingSlash, "/")
	cleanedNormalized := strings.Trim(cleanedNoLeadingSlash, "/")

	if originalNormalized != cleanedNormalized && originalNormalized != "" {
		// Allow the case where original is empty but cleaned is also effectively empty
		if !(originalNormalized == "." && cleanedNormalized == "") {
			return fmt.Errorf("subPath contains invalid path components: %s", subPath)
		}
	}

	// Check for null bytes which could be used for injection
	if strings.Contains(subPath, "\x00") {
		return fmt.Errorf("subPath contains null byte")
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
		// Validate subPath to prevent path traversal attacks
		if err := validateSubPath(subPath); err != nil {
			return "", "", fmt.Errorf("invalid subPath: %w", err)
		}
		// Combine share with subPath
		share = strings.TrimSuffix(share, "/") + "/" + strings.TrimPrefix(subPath, "/")
		klog.V(2).Infof("Combined NFS path: %s:%s (original share: %s, subPath: %s)",
			server, share, volumeContext[ParamShare], subPath)
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
	// Parse JSON-encoded annotations properly
	// Format: {"nfs.csi.takutakahashi.dev/subPath":"value",...}
	var annotations map[string]string
	if err := json.Unmarshal([]byte(annotationsJSON), &annotations); err != nil {
		klog.V(4).Infof("Failed to parse PVC annotations JSON: %v", err)
		return ""
	}

	subPath, ok := annotations[AnnotationSubPath]
	if !ok {
		return ""
	}

	return subPath
}
