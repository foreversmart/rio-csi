package driver

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
)

type RioCSI struct {
	name     string
	nodeID   string
	version  string
	endpoint string

	// Add CSI plugin parameters here
	enableIdentityServer   bool
	enableControllerServer bool
	enableNodeServer       bool

	cap   []*csi.VolumeCapability_AccessMode
	cscap []*csi.ControllerServiceCapability
}

func NewCSIDriver(name, version, nodeID, endpoint string, enableIdentityServer, enableControllerServer, enableNodeServer bool) *RioCSI {
	logrus.Infof("Driver: %s version: %s", name, version)

	// Add some check here
	//if parameter1 == "" {
	//	logrus.Fatal("parameter1 is empty")
	//}

	n := &RioCSI{
		name:                   name,
		nodeID:                 nodeID,
		version:                version,
		endpoint:               endpoint,
		enableIdentityServer:   enableControllerServer,
		enableControllerServer: enableControllerServer,
		enableNodeServer:       enableNodeServer,
	}

	// Add access modes for CSI here
	n.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	})

	// Add service capabilities for CSI here
	n.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
	})

	return n
}

func (n *RioCSI) Run() {
	var identityServer csi.IdentityServer
	var controllerServer csi.ControllerServer
	var nodeServer csi.NodeServer

	if n.enableIdentityServer {
		logrus.Info("Enable gRPC Server: IdentityServer")
		identityServer = NewIdentityServer(n)
	}
	if n.enableControllerServer {
		logrus.Info("Enable gRPC Server: ControllerServer")
		controllerServer = NewControllerServer(n)
	}
	if n.enableNodeServer {
		logrus.Info("Enable gRPC Server: NodeServer")
		nodeServer = NewNodeServer(n)
	}

	server := NewNonBlockingGRPCServer()
	server.Start(
		n.endpoint,
		identityServer,
		controllerServer,
		nodeServer,
	)
	server.Wait()
}

func (n *RioCSI) AddVolumeCapabilityAccessModes(vc []csi.VolumeCapability_AccessMode_Mode) {
	var vca []*csi.VolumeCapability_AccessMode
	for _, c := range vc {
		logrus.Infof("Enabling volume access mode: %v", c.String())
		vca = append(vca, &csi.VolumeCapability_AccessMode{Mode: c})
	}
	n.cap = vca
}

func (n *RioCSI) AddControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) {
	var csc []*csi.ControllerServiceCapability
	for _, c := range cl {
		logrus.Infof("Enabling controller service capability: %v", c.String())
		csc = append(csc, NewControllerServiceCapability(c))
	}
	n.cscap = csc
}
