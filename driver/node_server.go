package driver

import (
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/exec"
	"k8s.io/utils/mount"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/iscsi"
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
		logrus.Warningf("PodLVInfo could not be obtained for volume_id: %s, err = %v", req.VolumeId, err)
		logrus.Error(req.VolumeContext)
	}

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
			logrus.Error(err)
			return nil, err
		}
	} else {
		// mount on different nodes using iscsi
		connector := iscsi.Connector{
			VolumeName:    vol.Name,
			TargetIqn:     vol.Spec.IscsiTarget,
			TargetPortals: []string{vol.Spec.IscsiPortal},
			Lun:           vol.Spec.IscsiLun,
		}

		devicePath, err := connector.Connect()
		if err != nil {
			logrus.Error(err)
			return nil, err
		}

		if devicePath == "" {
			logrus.Error("connect reported success, but no path returned")
			return nil, fmt.Errorf("connect reported success, but no path returned")
		}

		mounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: exec.New()}
		err = mounter.FormatAndMount(devicePath, mountInfo.MountPath, mountInfo.FSType, mountInfo.MountOptions)
		if err != nil {
			logrus.Errorf("iscsi: failed to mount iscsi volume %s [%s] to %s, error %v", devicePath, mountInfo.FSType, mountInfo.MountPath, err)
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
	logrus.Infof("hostpath: volume %s path: %s has been unmounted.",
		volumeID, targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeGetVolumeStats(_ context.Context, _ *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	logrus.Debugf("running NodeGetVolumeStats...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented NodeGetVolumeStats")
}

func (ns *nodeServer) NodeUnstageVolume(_ context.Context, _ *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	logrus.Debugf("running NodeUnstageVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented NodeUnstageVolume")
}

func (ns *nodeServer) NodeStageVolume(_ context.Context, _ *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	logrus.Debugf("running NodeStageVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented NodeStageVolume")
}

func (ns *nodeServer) NodeExpandVolume(_ context.Context, _ *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	logrus.Debugf("running NodeExpandVolume...")
	return nil, status.Error(codes.Unimplemented, "Unimplemented NodeExpandVolume")
}

func (ns *nodeServer) NodeGetInfo(_ context.Context, _ *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	logrus.Infof("Using default NodeGetInfo")
	return &csi.NodeGetInfoResponse{
		NodeId:            ns.Driver.nodeID,
		MaxVolumesPerNode: 65535,
	}, nil
}

func (ns *nodeServer) NodeGetCapabilities(_ context.Context, _ *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	logrus.Infof("Using default NodeGetCapabilities")

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
