package nfs

import (
	"context"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

func TestGetPluginInfo(t *testing.T) {
	driverName := "test.csi.driver"
	driver, err := NewDriver(driverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	resp, err := driver.GetPluginInfo(context.Background(), &csi.GetPluginInfoRequest{})
	if err != nil {
		t.Fatalf("GetPluginInfo failed: %v", err)
	}

	if resp.Name != driverName {
		t.Errorf("Expected driver name %s, got %s", driverName, resp.Name)
	}

	if resp.VendorVersion != DriverVersion {
		t.Errorf("Expected version %s, got %s", DriverVersion, resp.VendorVersion)
	}
}

func TestGetPluginCapabilities(t *testing.T) {
	driver, err := NewDriver(DefaultDriverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	resp, err := driver.GetPluginCapabilities(context.Background(), &csi.GetPluginCapabilitiesRequest{})
	if err != nil {
		t.Fatalf("GetPluginCapabilities failed: %v", err)
	}

	// Check that CONTROLLER_SERVICE capability is advertised
	hasControllerService := false
	for _, cap := range resp.Capabilities {
		if svc := cap.GetService(); svc != nil {
			if svc.Type == csi.PluginCapability_Service_CONTROLLER_SERVICE {
				hasControllerService = true
			}
		}
	}

	if !hasControllerService {
		t.Error("Expected CONTROLLER_SERVICE capability")
	}
}

func TestProbe(t *testing.T) {
	driver, err := NewDriver(DefaultDriverName, "test-node", "unix:///tmp/test.sock")
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	resp, err := driver.Probe(context.Background(), &csi.ProbeRequest{})
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}

	if resp.Ready == nil || !resp.Ready.Value {
		t.Error("Expected driver to be ready")
	}
}
