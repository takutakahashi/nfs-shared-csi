package main

import (
	"flag"
	"os"

	"github.com/example/nfs-shared-csi/pkg/nfs"
	"k8s.io/klog/v2"
)

var (
	endpoint   = flag.String("endpoint", "unix:///csi/csi.sock", "CSI endpoint")
	nodeID     = flag.String("nodeid", "", "Node ID")
	driverName = flag.String("drivername", nfs.DefaultDriverName, "CSI driver name")
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	if *nodeID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			klog.Fatalf("Failed to get hostname: %v", err)
		}
		*nodeID = hostname
	}

	klog.Infof("Starting NFS CSI driver: %s, nodeID: %s, endpoint: %s", *driverName, *nodeID, *endpoint)

	driver, err := nfs.NewDriver(*driverName, *nodeID, *endpoint)
	if err != nil {
		klog.Fatalf("Failed to create driver: %v", err)
	}

	if err := driver.Run(); err != nil {
		klog.Fatalf("Failed to run driver: %v", err)
	}
}
