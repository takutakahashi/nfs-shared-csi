package nfs

import (
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

func TestValidateVolumeCapability(t *testing.T) {
	tests := []struct {
		name    string
		cap     *csi.VolumeCapability
		wantErr bool
	}{
		{
			name:    "nil capability",
			cap:     nil,
			wantErr: true,
		},
		{
			name:    "nil access mode",
			cap:     &csi.VolumeCapability{},
			wantErr: true,
		},
		{
			name: "nil access type",
			cap: &csi.VolumeCapability{
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
				},
			},
			wantErr: true,
		},
		{
			name: "block access type",
			cap: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Block{
					Block: &csi.VolumeCapability_BlockVolume{},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
				},
			},
			wantErr: true,
		},
		{
			name: "valid mount RWX",
			cap: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
				},
			},
			wantErr: false,
		},
		{
			name: "valid mount ROX",
			cap: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
				},
			},
			wantErr: false,
		},
		{
			name: "valid single node writer",
			cap: &csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVolumeCapability(tt.cap)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateVolumeCapability() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetVolumeSource(t *testing.T) {
	tests := []struct {
		name       string
		ctx        map[string]string
		wantServer string
		wantShare  string
		wantErr    bool
	}{
		{
			name:    "empty context",
			ctx:     map[string]string{},
			wantErr: true,
		},
		{
			name: "missing server",
			ctx: map[string]string{
				"share": "/data",
			},
			wantErr: true,
		},
		{
			name: "missing share",
			ctx: map[string]string{
				"server": "192.168.1.1",
			},
			wantErr: true,
		},
		{
			name: "valid with leading slash",
			ctx: map[string]string{
				"server": "192.168.1.1",
				"share":  "/data",
			},
			wantServer: "192.168.1.1",
			wantShare:  "/data",
			wantErr:    false,
		},
		{
			name: "valid without leading slash",
			ctx: map[string]string{
				"server": "192.168.1.1",
				"share":  "data",
			},
			wantServer: "192.168.1.1",
			wantShare:  "/data", // Should add leading slash
			wantErr:    false,
		},
		{
			name: "hostname server",
			ctx: map[string]string{
				"server": "nfs.example.com",
				"share":  "/exports/share",
			},
			wantServer: "nfs.example.com",
			wantShare:  "/exports/share",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, share, err := getVolumeSource(tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("getVolumeSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if server != tt.wantServer {
					t.Errorf("getVolumeSource() server = %v, want %v", server, tt.wantServer)
				}
				if share != tt.wantShare {
					t.Errorf("getVolumeSource() share = %v, want %v", share, tt.wantShare)
				}
			}
		})
	}
}
