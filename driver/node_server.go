package driver

import (
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/crd"
	iscsi2 "qiniu.io/rio-csi/lib/iscsi"
	mount2 "qiniu.io/rio-csi/lib/mount"
	"qiniu.io/rio-csi/logger"
	"sync"
)

type NodeServer struct {
	Driver *RioCSI
	Lock   sync.Mutex
	// Users add fields as needed.
	//
	// In the NFS CSI implementation, we need to mount the nfs server to the local,
	// so we need a mounter instance.
	//
	// In the CSI implementation of other storage vendors, you may need to add other
	// instances, such as the api client of Alibaba Cloud Storage.
	//mounter mount.Interface
}

func (ns *NodeServer) NodePublishVolume(_ context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	var (
		err error
	)

	if err = ns.validateNodePublishReq(req); err != nil {
		return nil, err
	}

	vol, mountInfo, err := GetVolAndMountInfo(req)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	podLVinfo, err := getPodLVInfo(req)
	if err != nil {
		logger.StdLog.Errorf("PodInfo could not be obtained for volume_id: %s, err = %v", req.VolumeId, err)
		logger.StdLog.Error(req.VolumeContext)
		return nil, status.Error(codes.Internal, err.Error())
	}

	// TODO vol and pod on the same node to local mount
	// check pod and vol on the same node
	//if podLVinfo.NodeId == vol.Spec.OwnerNodeID {
	//	// if on same node go directly lvm mount
	//	mountInfo.DevicePath = mount.GetVolumeDevicePath(vol)
	//} else {
	node, getErr := client.DefaultClient.InternalClientSet.RioV1().RioNodes(vol.Namespace).Get(context.TODO(), vol.Spec.OwnerNodeID, metav1.GetOptions{})
	if getErr != nil {
		logger.StdLog.Errorf("get %s rio node %s info error %v", vol.Namespace, vol.Spec.OwnerNodeID, getErr)
		return nil, getErr
	}

	// add lock to limit iscsi connector
	ns.Lock.Lock()
	defer ns.Lock.Unlock()

	// mount on different nodes using iscsi
	connector := iscsi2.Connector{
		AuthType:      "chap",
		VolumeName:    vol.Name,
		TargetIqn:     vol.Spec.IscsiTarget,
		TargetPortals: []string{node.ISCSIInfo.Portal},
		Lun:           vol.Spec.IscsiLun,
		DiscoverySecrets: iscsi2.Secrets{
			SecretsType: "chap",
			UserName:    ns.Driver.iscsiUsername,
			Password:    ns.Driver.iscsiPassword,
		},
		DoDiscovery:     true,
		DoCHAPDiscovery: true,
	}

	devicePath, connectErr := connector.Connect()
	if connectErr != nil {
		logger.StdLog.Error(connectErr)
		return nil, connectErr
	}

	if devicePath == "" {
		logger.StdLog.Error("connect reported success, but no path returned")
		return nil, fmt.Errorf("connect reported success, but no path returned")
	}

	mountInfo.DevicePath = devicePath

	vol.Spec.MountNodes = append(vol.Spec.MountNodes, node.Name)
	vol, err = crd.UpdateVolume(vol)
	if err != nil {
		logger.StdLog.Errorf("update volume %s mount nodes error %v", vol.Name, err)
		return nil, fmt.Errorf("update volume error %v", err)
	}

	logger.StdLog.Info("node publish volume", podLVinfo, mountInfo)

	switch req.GetVolumeCapability().GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		// attempt block mount operation on the requested path
		err = mount2.MountBlock(vol, mountInfo, podLVinfo)
	case *csi.VolumeCapability_Mount:
		// attempt filesystem mount operation on the requested path
		err = mount2.MountFilesystem(vol, mountInfo, podLVinfo)
	}

	if err != nil {
		logger.StdLog.Error(err)
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil

}

func (ns *NodeServer) NodeUnpublishVolume(_ context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	var (
		err error
		vol *apis.Volume
	)

	if err = ns.validateNodeUnPublishReq(req); err != nil {
		return nil, err
	}

	targetPath := req.GetTargetPath()
	volumeID := req.GetVolumeId()

	if vol, err = crd.GetVolume(volumeID); err != nil {
		return nil, status.Errorf(codes.Internal,
			"not able to get the LVMVolume %s err : %s",
			volumeID, err.Error())
	}

	newMountNodes := make([]string, 0, 5)
	isRemoved := false
	for _, v := range vol.Spec.MountNodes {
		if v == ns.Driver.nodeID && !isRemoved {
			isRemoved = true
			continue
		}

		newMountNodes = append(newMountNodes, v)
	}

	vol.Spec.MountNodes = newMountNodes

	vol, err = crd.UpdateVolume(vol)
	if err != nil {
		logger.StdLog.Errorf("update volume %s mount nodes error %v", vol.Name, err)
		return nil, fmt.Errorf("update volume error %v", err)
	}

	err = mount2.UmountVolume(vol, targetPath)

	if err != nil {
		return nil, status.Errorf(codes.Internal,
			"unable to umount the volume %s err : %s",
			volumeID, err.Error())
	}
	logger.StdLog.Infof("hostpath: volume %s path: %s has been unmounted.",
		volumeID, targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *NodeServer) NodeGetVolumeStats(_ context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {

	volID := req.GetVolumeId()
	path := req.GetVolumePath()
	logger.StdLog.Infof("receive request NodeGetVolumeStats volume id: %s, path: %s", volID, path)

	if len(volID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id is not provided")
	}
	if len(path) == 0 {
		return nil, status.Error(codes.InvalidArgument, "path is not provided")
	}

	if !mount2.IsMountPath(path) {
		return nil, status.Error(codes.NotFound, "path is not a mount path")
	}

	var sfs unix.Statfs_t
	if err := unix.Statfs(path, &sfs); err != nil {
		return nil, status.Errorf(codes.Internal, "statfs on %s failed: %v", path, err)
	}

	var usage []*csi.VolumeUsage
	usage = append(usage, &csi.VolumeUsage{
		Unit:      csi.VolumeUsage_BYTES,
		Total:     int64(sfs.Blocks) * int64(sfs.Bsize),
		Used:      int64(sfs.Blocks-sfs.Bfree) * int64(sfs.Bsize),
		Available: int64(sfs.Bavail) * int64(sfs.Bsize),
	})
	usage = append(usage, &csi.VolumeUsage{
		Unit:      csi.VolumeUsage_INODES,
		Total:     int64(sfs.Files),
		Used:      int64(sfs.Files - sfs.Ffree),
		Available: int64(sfs.Ffree),
	})

	return &csi.NodeGetVolumeStatsResponse{Usage: usage}, nil
}

func (ns *NodeServer) NodeUnstageVolume(_ context.Context, _ *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	logger.StdLog.Debugf("running NodeUnstageVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented NodeUnstageVolume")
}

func (ns *NodeServer) NodeStageVolume(_ context.Context, _ *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	logger.StdLog.Debugf("running NodeStageVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented NodeStageVolume")
}

func (ns *NodeServer) NodeExpandVolume(_ context.Context, _ *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	logger.StdLog.Debugf("running NodeExpandVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented NodeExpandVolume")
}

func (ns *NodeServer) NodeGetInfo(_ context.Context, _ *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	logger.StdLog.Infof("Using default NodeGetInfo")
	return &csi.NodeGetInfoResponse{
		NodeId:            ns.Driver.nodeID,
		MaxVolumesPerNode: 65535,
	}, nil
}

func (ns *NodeServer) NodeGetCapabilities(_ context.Context, _ *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	logger.StdLog.Infof("Using default NodeGetCapabilities")

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
					},
				},
			},
		},
	}, nil
}
