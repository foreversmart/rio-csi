package driver

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"qiniu.io/rio-csi/logger"
)

type RioCSI struct {
	name     string
	nodeID   string
	version  string
	endpoint string

	iscsiUsername string
	iscsiPassword string

	// Add CSI plugin parameters here
	enableIdentityServer   bool
	enableControllerServer bool
	enableNodeServer       bool

	accessModes         []*csi.VolumeCapability_AccessMode
	serviceCapabilities []*csi.ControllerServiceCapability
}

func NewCSIDriver(name, version, nodeID, endpoint, iscsiUsername, iscsiPassword string, enableIdentityServer, enableControllerServer, enableNodeServer bool) *RioCSI {
	logger.StdLog.Infof("Driver: %s version: %s", name, version)

	// Add some check here
	//if parameter1 == "" {
	//	logger.StdLog.Fatal("parameter1 is empty")
	//}

	n := &RioCSI{
		name:                   name,
		nodeID:                 nodeID,
		version:                version,
		endpoint:               endpoint,
		iscsiUsername:          iscsiUsername,
		iscsiPassword:          iscsiPassword,
		enableIdentityServer:   enableIdentityServer,
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
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
		//csi.ControllerServiceCapability_RPC_GET_CAPACITY,
	})

	return n
}

func (n *RioCSI) Run() {
	var identityServer csi.IdentityServer
	var controllerServer csi.ControllerServer
	var nodeServer csi.NodeServer

	if n.enableIdentityServer {
		logger.StdLog.Info("Enable gRPC Server: IdentityServer")
		identityServer = NewIdentityServer(n)
	}

	if n.enableControllerServer {
		logger.StdLog.Info("Enable gRPC Server: ControllerServer")
		controllerServer = NewControllerServer(n)
	}
	if n.enableNodeServer {
		logger.StdLog.Info("Enable gRPC Server: NodeServer")
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
		logger.StdLog.Infof("Enabling volume access mode: %v", c.String())
		vca = append(vca, &csi.VolumeCapability_AccessMode{Mode: c})
	}
	n.accessModes = vca
}

func (n *RioCSI) AddControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) {
	var csc []*csi.ControllerServiceCapability
	for _, c := range cl {
		logger.StdLog.Infof("Enabling controller service capability: %v", c.String())
		csc = append(csc, NewControllerServiceCapability(c))
	}
	n.serviceCapabilities = csc
}
