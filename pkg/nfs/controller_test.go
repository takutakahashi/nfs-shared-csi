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

	// Dynamic provisioning - should have CREATE_DELETE_VOLUME capability
	if len(resp.Capabilities) != 1 {
		t.Errorf("Expected 1 capability for dynamic provisioning, got %d", len(resp.Capabilities))
	}

	if len(resp.Capabilities) > 0 {
		cap := resp.Capabilities[0]
		if cap.GetRpc().GetType() != csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME {
			t.Errorf("Expected CREATE_DELETE_VOLUME capability, got %v", cap.GetRpc().GetType())
		}
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

func TestCreateVolume(t *testing.T) {
	driver, err := NewDriver(DefaultDriverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	tests := []struct {
		name     string
		req      *csi.CreateVolumeRequest
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name:     "missing volume name",
			req:      &csi.CreateVolumeRequest{},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing volume capabilities",
			req: &csi.CreateVolumeRequest{
				Name: "test-volume",
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing server parameter",
			req: &csi.CreateVolumeRequest{
				Name: "test-volume",
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
				Parameters: map[string]string{
					"share": "/exports/data",
				},
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing share parameter",
			req: &csi.CreateVolumeRequest{
				Name: "test-volume",
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
				Parameters: map[string]string{
					"server": "192.168.1.100",
				},
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "valid create volume request",
			req: &csi.CreateVolumeRequest{
				Name: "test-volume",
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
				Parameters: map[string]string{
					"server": "192.168.1.100",
					"share":  "/exports/data",
				},
			},
			wantErr: false,
		},
		{
			name: "valid create volume request with subPath",
			req: &csi.CreateVolumeRequest{
				Name: "test-volume",
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
				Parameters: map[string]string{
					"server":  "192.168.1.100",
					"share":   "/exports/data",
					"subPath": "myapp",
				},
			},
			wantErr: false,
		},
		{
			name: "valid create volume request with subPath from PVC annotation",
			req: &csi.CreateVolumeRequest{
				Name: "test-volume",
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
				Parameters: map[string]string{
					"server":                               "192.168.1.100",
					"share":                                "/exports/data",
					"csi.storage.k8s.io/pvc/annotations":   `{"nfs.csi.takutakahashi.dev/subPath":"music"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "StorageClass subPath takes priority over PVC annotation",
			req: &csi.CreateVolumeRequest{
				Name: "test-volume",
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
				Parameters: map[string]string{
					"server":                               "192.168.1.100",
					"share":                                "/exports/data",
					"subPath":                              "priority-path",
					"csi.storage.k8s.io/pvc/annotations":   `{"nfs.csi.takutakahashi.dev/subPath":"music"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid subPath from PVC annotation",
			req: &csi.CreateVolumeRequest{
				Name: "test-volume",
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
				Parameters: map[string]string{
					"server":                               "192.168.1.100",
					"share":                                "/exports/data",
					"csi.storage.k8s.io/pvc/annotations":   `{"nfs.csi.takutakahashi.dev/subPath":"../../etc/passwd"}`,
				},
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := driver.CreateVolume(context.Background(), tt.req)

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

			if resp.Volume.VolumeId == "" {
				t.Error("Expected volume ID to be set")
			}

			if resp.Volume.VolumeContext["server"] != tt.req.Parameters["server"] {
				t.Errorf("Expected server %s, got %s", tt.req.Parameters["server"], resp.Volume.VolumeContext["server"])
			}

			if resp.Volume.VolumeContext["share"] != tt.req.Parameters["share"] {
				t.Errorf("Expected share %s, got %s", tt.req.Parameters["share"], resp.Volume.VolumeContext["share"])
			}

			// Check subPath if it's in parameters (StorageClass)
			if subPath, ok := tt.req.Parameters["subPath"]; ok {
				if resp.Volume.VolumeContext["subPath"] != subPath {
					t.Errorf("Expected subPath %s, got %s", subPath, resp.Volume.VolumeContext["subPath"])
				}
			}

			// Special case: check subPath from PVC annotation
			if tt.name == "valid create volume request with subPath from PVC annotation" {
				if resp.Volume.VolumeContext["subPath"] != "music" {
					t.Errorf("Expected subPath 'music' from PVC annotation, got %s", resp.Volume.VolumeContext["subPath"])
				}
			}

			// Special case: check priority (StorageClass > PVC annotation)
			if tt.name == "StorageClass subPath takes priority over PVC annotation" {
				if resp.Volume.VolumeContext["subPath"] != "priority-path" {
					t.Errorf("Expected subPath 'priority-path' from StorageClass parameter, got %s", resp.Volume.VolumeContext["subPath"])
				}
			}
		})
	}
}

func TestDeleteVolume(t *testing.T) {
	driver, err := NewDriver(DefaultDriverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	tests := []struct {
		name     string
		req      *csi.DeleteVolumeRequest
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name:     "missing volume ID",
			req:      &csi.DeleteVolumeRequest{},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "valid delete volume request",
			req: &csi.DeleteVolumeRequest{
				VolumeId: "test-volume",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := driver.DeleteVolume(context.Background(), tt.req)

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

			if resp == nil {
				t.Error("Expected response to be non-nil")
			}
		})
	}
}
