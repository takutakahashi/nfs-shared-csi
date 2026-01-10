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
		{
			name: "with subPath parameter",
			ctx: map[string]string{
				"server":  "192.168.1.1",
				"share":   "/data",
				"subPath": "app1",
			},
			wantServer: "192.168.1.1",
			wantShare:  "/data/app1",
			wantErr:    false,
		},
		{
			name: "with subPath with leading slash",
			ctx: map[string]string{
				"server":  "192.168.1.1",
				"share":   "/data",
				"subPath": "/app2",
			},
			wantServer: "192.168.1.1",
			wantShare:  "/data/app2",
			wantErr:    false,
		},
		{
			name: "with subPath and share trailing slash",
			ctx: map[string]string{
				"server":  "192.168.1.1",
				"share":   "/data/",
				"subPath": "app3",
			},
			wantServer: "192.168.1.1",
			wantShare:  "/data/app3",
			wantErr:    false,
		},
		{
			name: "with nested subPath",
			ctx: map[string]string{
				"server":  "nfs.example.com",
				"share":   "/exports",
				"subPath": "tenant1/data",
			},
			wantServer: "nfs.example.com",
			wantShare:  "/exports/tenant1/data",
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

func TestGetSubPath(t *testing.T) {
	tests := []struct {
		name string
		ctx  map[string]string
		want string
	}{
		{
			name: "no subPath",
			ctx:  map[string]string{},
			want: "",
		},
		{
			name: "subPath from parameter",
			ctx: map[string]string{
				"subPath": "mypath",
			},
			want: "mypath",
		},
		{
			name: "subPath from PVC annotation",
			ctx: map[string]string{
				"csi.storage.k8s.io/pvc/annotations": `{"nfs.csi.example.com/subPath":"annotated-path"}`,
			},
			want: "annotated-path",
		},
		{
			name: "parameter takes priority over annotation",
			ctx: map[string]string{
				"subPath": "param-path",
				"csi.storage.k8s.io/pvc/annotations": `{"nfs.csi.example.com/subPath":"annotated-path"}`,
			},
			want: "param-path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSubPath(tt.ctx)
			if got != tt.want {
				t.Errorf("getSubPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAnnotationSubPath(t *testing.T) {
	tests := []struct {
		name            string
		annotationsJSON string
		want            string
	}{
		{
			name:            "empty string",
			annotationsJSON: "",
			want:            "",
		},
		{
			name:            "no subPath annotation",
			annotationsJSON: `{"other.annotation":"value"}`,
			want:            "",
		},
		{
			name:            "valid subPath annotation",
			annotationsJSON: `{"nfs.csi.example.com/subPath":"mypath"}`,
			want:            "mypath",
		},
		{
			name:            "subPath annotation with other annotations",
			annotationsJSON: `{"foo":"bar","nfs.csi.example.com/subPath":"mypath","baz":"qux"}`,
			want:            "mypath",
		},
		{
			name:            "nested path",
			annotationsJSON: `{"nfs.csi.example.com/subPath":"tenant1/app1/data"}`,
			want:            "tenant1/app1/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAnnotationSubPath(tt.annotationsJSON)
			if got != tt.want {
				t.Errorf("parseAnnotationSubPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
