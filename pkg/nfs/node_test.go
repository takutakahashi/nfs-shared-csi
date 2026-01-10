package nfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNodePublishVolume_Validation(t *testing.T) {
	driver, err := NewDriver(DefaultDriverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	tests := []struct {
		name    string
		req     *csi.NodePublishVolumeRequest
		wantErr codes.Code
	}{
		{
			name:    "missing volume ID",
			req:     &csi.NodePublishVolumeRequest{},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "missing target path",
			req: &csi.NodePublishVolumeRequest{
				VolumeId: "test-volume",
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "missing volume capability",
			req: &csi.NodePublishVolumeRequest{
				VolumeId:   "test-volume",
				TargetPath: "/tmp/target",
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "missing server parameter",
			req: &csi.NodePublishVolumeRequest{
				VolumeId:   "test-volume",
				TargetPath: "/tmp/target",
				VolumeCapability: &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
					},
				},
				VolumeContext: map[string]string{},
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "missing share parameter",
			req: &csi.NodePublishVolumeRequest{
				VolumeId:   "test-volume",
				TargetPath: "/tmp/target",
				VolumeCapability: &csi.VolumeCapability{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
					},
				},
				VolumeContext: map[string]string{
					"server": "192.168.1.1",
				},
			},
			wantErr: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := driver.NodePublishVolume(context.Background(), tt.req)
			if err == nil {
				t.Errorf("Expected error, got nil")
				return
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Errorf("Expected gRPC status error, got %v", err)
				return
			}
			if st.Code() != tt.wantErr {
				t.Errorf("Expected error code %v, got %v", tt.wantErr, st.Code())
			}
		})
	}
}

func TestNodeUnpublishVolume_Validation(t *testing.T) {
	driver, err := NewDriver(DefaultDriverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	tests := []struct {
		name    string
		req     *csi.NodeUnpublishVolumeRequest
		wantErr codes.Code
	}{
		{
			name:    "missing volume ID",
			req:     &csi.NodeUnpublishVolumeRequest{},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "missing target path",
			req: &csi.NodeUnpublishVolumeRequest{
				VolumeId: "test-volume",
			},
			wantErr: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := driver.NodeUnpublishVolume(context.Background(), tt.req)
			if err == nil {
				t.Errorf("Expected error, got nil")
				return
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Errorf("Expected gRPC status error, got %v", err)
				return
			}
			if st.Code() != tt.wantErr {
				t.Errorf("Expected error code %v, got %v", tt.wantErr, st.Code())
			}
		})
	}
}

func TestNodeUnpublishVolume_NonExistentPath(t *testing.T) {
	driver, err := NewDriver(DefaultDriverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "csi-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	nonExistentPath := filepath.Join(tmpDir, "non-existent")

	req := &csi.NodeUnpublishVolumeRequest{
		VolumeId:   "test-volume",
		TargetPath: nonExistentPath,
	}

	// Should succeed even if path doesn't exist
	_, err = driver.NodeUnpublishVolume(context.Background(), req)
	if err != nil {
		t.Errorf("Expected no error for non-existent path, got %v", err)
	}
}

func TestNodeGetInfo(t *testing.T) {
	nodeID := "test-node-123"
	driver, err := NewDriver(DefaultDriverName, nodeID, "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	resp, err := driver.NodeGetInfo(context.Background(), &csi.NodeGetInfoRequest{})
	if err != nil {
		t.Fatalf("NodeGetInfo failed: %v", err)
	}

	if resp.NodeId != nodeID {
		t.Errorf("Expected node ID %s, got %s", nodeID, resp.NodeId)
	}
}

func TestNodeGetCapabilities(t *testing.T) {
	driver, err := NewDriver(DefaultDriverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	resp, err := driver.NodeGetCapabilities(context.Background(), &csi.NodeGetCapabilitiesRequest{})
	if err != nil {
		t.Fatalf("NodeGetCapabilities failed: %v", err)
	}

	// Our driver doesn't advertise any special capabilities
	if len(resp.Capabilities) != 0 {
		t.Errorf("Expected 0 capabilities, got %d", len(resp.Capabilities))
	}
}
