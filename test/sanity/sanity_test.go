// Package sanity provides CSI sanity tests for the NFS driver.
//
// NOTE: This driver implements static provisioning only.
// Many CSI sanity tests will fail because they're designed for
// dynamic provisioning drivers (CreateVolume, DeleteVolume, Snapshots, etc.).
// These tests are useful for local development and debugging of Identity
// and Node service implementations.
package sanity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kubernetes-csi/csi-test/v5/pkg/sanity"
	"k8s.io/mount-utils"

	"github.com/example/nfs-shared-csi/pkg/nfs"
)

func TestSanity(t *testing.T) {
	// Create temporary directories for testing
	tmpDir, err := os.MkdirTemp("", "csi-sanity")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	endpoint := filepath.Join(tmpDir, "csi.sock")
	targetPath := filepath.Join(tmpDir, "target")
	stagingPath := filepath.Join(tmpDir, "staging")

	// Use fake mounter for testing without actual NFS
	fakeMounter := mount.NewFakeMounter([]mount.MountPoint{})

	// Create the driver with fake mounter
	driver, err := nfs.NewDriver(
		nfs.DefaultDriverName,
		"test-node",
		"unix://"+endpoint,
		nfs.WithMounter(fakeMounter),
	)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	// Start the driver in a goroutine
	go func() {
		if err := driver.Run(); err != nil {
			t.Logf("Driver exited: %v", err)
		}
	}()
	defer driver.Stop()

	// Configure sanity test
	config := sanity.NewTestConfig()
	config.Address = endpoint
	config.TargetPath = targetPath
	config.StagingPath = stagingPath
	config.TestVolumeParameters = map[string]string{
		"server": "localhost",
		"share":  "/test",
	}
	config.TestVolumeAccessType = "mount"
	config.IDGen = &sanity.DefaultIDGenerator{}

	// Skip tests for features we don't support (static provisioning only)
	config.TestNodeVolumeAttachLimit = false

	// Run sanity tests
	// Skip snapshot and volume expansion tests as we don't support them
	sanity.Test(t, config)
}
