package driver

import (
	"fmt"
	"k8s.io/utils/mount"
	"qiniu.io/rio-csi/driver/scheduler"
	"qiniu.io/rio-csi/logger"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func NewIdentityServer(d *RioCSI) *IdentityServer {
	return &IdentityServer{
		Driver: d,
	}
}

func NewControllerServer(d *RioCSI) *ControllerServer {
	return &ControllerServer{
		Driver:           d,
		mounter:          mount.New(""),
		schedulerManager: scheduler.NewManager(),
	}
}

func NewNodeServer(n *RioCSI) *NodeServer {
	return &NodeServer{
		Driver: n,
	}
}

func NewControllerServiceCapability(cap csi.ControllerServiceCapability_RPC_Type) *csi.ControllerServiceCapability {
	return &csi.ControllerServiceCapability{
		Type: &csi.ControllerServiceCapability_Rpc{
			Rpc: &csi.ControllerServiceCapability_RPC{
				Type: cap,
			},
		},
	}
}

func ParseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	logger.StdLog.Infof("GRPC call: %s", info.FullMethod)
	logger.StdLog.Infof("GRPC request: %s", protosanitizer.StripSecrets(req))
	resp, err := handler(ctx, req)
	if err != nil {
		logger.StdLog.Errorf("GRPC error: %v", err)
	} else {
		logger.StdLog.Infof("GRPC response: %s", protosanitizer.StripSecrets(resp))
	}
	return resp, err
}
