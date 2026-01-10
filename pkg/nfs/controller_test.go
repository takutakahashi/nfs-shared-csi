package nfs

import (
	"context"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestControllerGetCapabilities(t *testing.T) {
	driver, err := NewDriver(DefaultDriverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	resp, err := driver.ControllerGetCapabilities(context.Background(), &csi.ControllerGetCapabilitiesRequest{})
	if err != nil {
		t.Fatalf("ControllerGetCapabilities failed: %v", err)
	}

	// Static provisioning only - no controller capabilities
	if len(resp.Capabilities) != 0 {
		t.Errorf("Expected 0 capabilities for static provisioning, got %d", len(resp.Capabilities))
	}
}

func TestValidateVolumeCapabilities(t *testing.T) {
	driver, err := NewDriver(DefaultDriverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	tests := []struct {
		name      string
		req       *csi.ValidateVolumeCapabilitiesRequest
		wantErr   bool
		wantCode  codes.Code
		confirmed bool
	}{
		{
			name:     "missing volume ID",
			req:      &csi.ValidateVolumeCapabilitiesRequest{},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing capabilities",
			req: &csi.ValidateVolumeCapabilitiesRequest{
				VolumeId: "test-volume",
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "valid RWX capability",
			req: &csi.ValidateVolumeCapabilitiesRequest{
				VolumeId: "test-volume",
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
				},
			},
			wantErr:   false,
			confirmed: true,
		},
		{
			name: "valid ROX capability",
			req: &csi.ValidateVolumeCapabilitiesRequest{
				VolumeId: "test-volume",
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
						},
					},
				},
			},
			wantErr:   false,
			confirmed: true,
		},
		{
			name: "valid single node writer",
			req: &csi.ValidateVolumeCapabilitiesRequest{
				VolumeId: "test-volume",
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessType: &csi.VolumeCapability_Mount{
							Mount: &csi.VolumeCapability_MountVolume{},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
						},
					},
				},
			},
			wantErr:   false,
			confirmed: true,
		},
		{
			name: "block access type not supported",
			req: &csi.ValidateVolumeCapabilitiesRequest{
				VolumeId: "test-volume",
				VolumeCapabilities: []*csi.VolumeCapability{
					{
						AccessType: &csi.VolumeCapability_Block{
							Block: &csi.VolumeCapability_BlockVolume{},
						},
						AccessMode: &csi.VolumeCapability_AccessMode{
							Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
						},
					},
				},
			},
			wantErr:   false,
			confirmed: false, // Block is not supported, so not confirmed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := driver.ValidateVolumeCapabilities(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("Expected gRPC status error, got %v", err)
					return
				}
				if st.Code() != tt.wantCode {
					t.Errorf("Expected error code %v, got %v", tt.wantCode, st.Code())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.confirmed {
				if resp.Confirmed == nil {
					t.Error("Expected confirmed capabilities")
				}
			} else {
				if resp.Confirmed != nil {
					t.Error("Expected capabilities to not be confirmed")
				}
			}
		})
	}
}

func TestCreateVolume_Unimplemented(t *testing.T) {
	driver, err := NewDriver(DefaultDriverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	_, err = driver.CreateVolume(context.Background(), &csi.CreateVolumeRequest{})
	if err == nil {
		t.Error("Expected error for unimplemented CreateVolume")
		return
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("Expected gRPC status error, got %v", err)
		return
	}

	if st.Code() != codes.Unimplemented {
		t.Errorf("Expected Unimplemented error, got %v", st.Code())
	}
}

func TestDeleteVolume_Unimplemented(t *testing.T) {
	driver, err := NewDriver(DefaultDriverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	_, err = driver.DeleteVolume(context.Background(), &csi.DeleteVolumeRequest{})
	if err == nil {
		t.Error("Expected error for unimplemented DeleteVolume")
		return
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("Expected gRPC status error, got %v", err)
		return
	}

	if st.Code() != codes.Unimplemented {
		t.Errorf("Expected Unimplemented error, got %v", st.Code())
	}
}
