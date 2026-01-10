package sanity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kubernetes-csi/csi-test/v5/pkg/sanity"

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

	// Create the driver
	driver, err := nfs.NewDriver(nfs.DefaultDriverName, "test-node", "unix://"+endpoint)
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
	// Skip tests that require actual NFS mount
	config.TestVolumeAccessType = "mount"
	config.IDGen = &sanity.DefaultIDGenerator{}

	// Run sanity tests
	sanity.Test(t, config)
}
