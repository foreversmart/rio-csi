package driver

import (
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/exec"
	"k8s.io/utils/mount"
	"os"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/iscsi"
	"qiniu.io/rio-csi/logger"
	"qiniu.io/rio-csi/lvm"
)

type nodeServer struct {
	Driver *RioCSI
	// Users add fields as needed.
	//
	// In the NFS CSI implementation, we need to mount the nfs server to the local,
	// so we need a mounter instance.
	//
	// In the CSI implementation of other storage vendors, you may need to add other
	// instances, such as the api client of Alibaba Cloud Storage.
	mounter mount.Interface
}

func (ns *nodeServer) NodePublishVolume(_ context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
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
		logger.StdLog.Errorf("PodLVInfo could not be obtained for volume_id: %s, err = %v", req.VolumeId, err)
		logger.StdLog.Error(req.VolumeContext)
		return nil, status.Error(codes.Internal, err.Error())
	}

	logger.StdLog.Info("node publish volume", podLVinfo)

	// check pod and vol on the same node
	if podLVinfo.NodeId == vol.Spec.OwnerNodeID {
		// if on same node go directly lvm mount
		switch req.GetVolumeCapability().GetAccessType().(type) {
		case *csi.VolumeCapability_Block:
			// attempt block mount operation on the requested path
			err = lvm.MountBlock(vol, mountInfo, podLVinfo)
		case *csi.VolumeCapability_Mount:
			// attempt filesystem mount operation on the requested path
			err = lvm.MountFilesystem(vol, mountInfo, podLVinfo)
		}

		if err != nil {
			logger.StdLog.Error(err)
			return nil, err
		}
	} else {
		node, getErr := client.DefaultClient.InternalClientSet.RioV1().RioNodes(vol.Namespace).Get(context.TODO(), vol.Spec.OwnerNodeID, metav1.GetOptions{})
		if getErr != nil {
			logger.StdLog.Errorf("get %s rio node %s info error %v", vol.Namespace, vol.Spec.OwnerNodeID, err)
			return nil, err
		}

		// mount on different nodes using iscsi
		connector := iscsi.Connector{
			AuthType:      "chap",
			VolumeName:    vol.Name,
			TargetIqn:     vol.Spec.IscsiTarget,
			TargetPortals: []string{node.ISCSIInfo.Portal},
			Lun:           vol.Spec.IscsiLun,
			DiscoverySecrets: iscsi.Secrets{
				SecretsType: "chap",
				UserName:    ns.Driver.iscsiUsername,
				Password:    ns.Driver.iscsiPassword,
			},
			DoDiscovery:     true,
			DoCHAPDiscovery: true,
		}

		devicePath, connectErr := connector.Connect()
		if connectErr != nil {
			logger.StdLog.Error(err)
			return nil, err
		}

		if devicePath == "" {
			logger.StdLog.Error("connect reported success, but no path returned")
			return nil, fmt.Errorf("connect reported success, but no path returned")
		}

		mntPath := mountInfo.MountPath
		mounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: exec.New()}

		// check mount point
		notMnt, mountErr := mounter.IsLikelyNotMountPoint(mntPath)
		if mountErr != nil && !os.IsNotExist(mountErr) {
			logger.StdLog.Errorf("heuristic determination of mount point failed %s error %v", mntPath, mountErr)
			return nil, fmt.Errorf("heuristic determination of mount point failed:%v", err)
		}
		if !notMnt {
			logger.StdLog.Infof("iscsi: %s already mounted", mntPath)
			return nil, nil
		}

		if err = os.MkdirAll(mntPath, 0o750); err != nil {
			logger.StdLog.Errorf("iscsi: failed to mkdir %s, error", mntPath)
			return nil, err
		}

		err = mounter.FormatAndMount(devicePath, mntPath, mountInfo.FSType, mountInfo.MountOptions)
		if err != nil {
			logger.StdLog.Errorf("iscsi: failed to mount iscsi volume %s [%s] to %s, error %v", devicePath, mountInfo.FSType, mntPath, err)
		}

		// TODO set IO limits
	}

	return &csi.NodePublishVolumeResponse{}, nil

}

func (ns *nodeServer) NodeUnpublishVolume(_ context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	var (
		err error
		vol *apis.Volume
	)

	if err = ns.validateNodeUnPublishReq(req); err != nil {
		return nil, err
	}

	targetPath := req.GetTargetPath()
	volumeID := req.GetVolumeId()

	if vol, err = lvm.GetVolume(volumeID); err != nil {
		return nil, status.Errorf(codes.Internal,
			"not able to get the LVMVolume %s err : %s",
			volumeID, err.Error())
	}

	err = lvm.UmountVolume(vol, targetPath)

	if err != nil {
		return nil, status.Errorf(codes.Internal,
			"unable to umount the volume %s err : %s",
			volumeID, err.Error())
	}
	logger.StdLog.Infof("hostpath: volume %s path: %s has been unmounted.",
		volumeID, targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeGetVolumeStats(_ context.Context, _ *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	logger.StdLog.Debugf("running NodeGetVolumeStats...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented NodeGetVolumeStats")
}

func (ns *nodeServer) NodeUnstageVolume(_ context.Context, _ *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	logger.StdLog.Debugf("running NodeUnstageVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented NodeUnstageVolume")
}

func (ns *nodeServer) NodeStageVolume(_ context.Context, _ *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	logger.StdLog.Debugf("running NodeStageVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented NodeStageVolume")
}

func (ns *nodeServer) NodeExpandVolume(_ context.Context, _ *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	logger.StdLog.Debugf("running NodeExpandVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented NodeExpandVolume")
}

func (ns *nodeServer) NodeGetInfo(_ context.Context, _ *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	logger.StdLog.Infof("Using default NodeGetInfo")
	return &csi.NodeGetInfoResponse{
		NodeId:            ns.Driver.nodeID,
		MaxVolumesPerNode: 65535,
	}, nil
}

func (ns *nodeServer) NodeGetCapabilities(_ context.Context, _ *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	logger.StdLog.Infof("Using default NodeGetCapabilities")

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_UNKNOWN,
					},
				},
			},
		},
	}, nil
}
