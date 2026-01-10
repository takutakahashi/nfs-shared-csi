package nfs

import (
	"net"
	"net/url"
	"os"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"k8s.io/mount-utils"
)

const (
	DefaultDriverName = "nfs.csi.example.com"
	DriverVersion     = "1.0.0"

	// Volume context keys
	ParamServer  = "server"
	ParamShare   = "share"
	ParamSubPath = "subPath"

	// PVC annotation key for subPath
	AnnotationSubPath = "nfs.csi.example.com/subPath"
)

type Driver struct {
	csi.UnimplementedIdentityServer
	csi.UnimplementedNodeServer
	csi.UnimplementedControllerServer

	name     string
	nodeID   string
	endpoint string
	version  string

	srv     *grpc.Server
	mounter mount.Interface

	mu sync.Mutex
}

// DriverOption is a functional option for configuring the driver
type DriverOption func(*Driver)

// WithMounter sets a custom mounter (useful for testing)
func WithMounter(m mount.Interface) DriverOption {
	return func(d *Driver) {
		d.mounter = m
	}
}

func NewDriver(name, nodeID, endpoint string, opts ...DriverOption) (*Driver, error) {
	klog.Infof("Creating new NFS CSI driver: name=%s, nodeID=%s", name, nodeID)

	d := &Driver{
		name:     name,
		nodeID:   nodeID,
		endpoint: endpoint,
		version:  DriverVersion,
		mounter:  mount.New(""),
	}

	for _, opt := range opts {
		opt(d)
	}

	return d, nil
}

func (d *Driver) Run() error {
	u, err := url.Parse(d.endpoint)
	if err != nil {
		return err
	}

	var addr string
	if u.Scheme == "unix" {
		addr = u.Path
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else {
		addr = u.Host
	}

	listener, err := net.Listen(u.Scheme, addr)
	if err != nil {
		return err
	}

	d.srv = grpc.NewServer(grpc.UnaryInterceptor(logGRPC))

	csi.RegisterIdentityServer(d.srv, d)
	csi.RegisterNodeServer(d.srv, d)
	csi.RegisterControllerServer(d.srv, d)

	klog.Infof("Listening on %s", d.endpoint)
	return d.srv.Serve(listener)
}

func (d *Driver) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.srv != nil {
		d.srv.GracefulStop()
	}
}
