package driver

import (
	"context"
	"errors"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/crd"
	"qiniu.io/rio-csi/mount"
	"strings"
)

func (ns *NodeServer) validateNodePublishReq(req *csi.NodePublishVolumeRequest) error {
	if req.GetVolumeCapability() == nil {
		return status.Error(codes.InvalidArgument,
			"Volume capability missing in request")
	}

	if len(req.GetVolumeId()) == 0 {
		return status.Error(codes.InvalidArgument,
			"Volume ID missing in request")
	}
	return nil
}

func (ns *NodeServer) validateNodeUnPublishReq(req *csi.NodeUnpublishVolumeRequest) error {
	if req.GetVolumeId() == "" {
		return status.Error(codes.InvalidArgument,
			"Volume ID missing in request")
	}

	if req.GetTargetPath() == "" {
		return status.Error(codes.InvalidArgument,
			"Target path missing in request")
	}
	return nil
}

// GetVolAndMountInfo get volume and mount info from node csi volume request
func GetVolAndMountInfo(req *csi.NodePublishVolumeRequest) (*apis.Volume, *mount.Info, error) {
	var info mount.Info

	info.FSType = req.GetVolumeCapability().GetMount().GetFsType()
	info.MountPath = req.GetTargetPath()
	info.MountOptions = append(info.MountOptions, req.GetVolumeCapability().GetMount().GetMountFlags()...)

	if req.GetReadonly() {
		info.MountOptions = append(info.MountOptions, "ro")
	} else {
		info.MountOptions = append(info.MountOptions, "rw")
	}

	volName := strings.ToLower(req.GetVolumeId())

	vol, err := crd.GetVolume(volName)
	if err != nil {
		return nil, nil, err
	}

	return vol, &info, nil
}

func getPodLVInfo(req *csi.NodePublishVolumeRequest) (*mount.PodInfo, error) {
	var podLVInfo mount.PodInfo
	var ok bool
	if podLVInfo.Name, ok = req.VolumeContext["csi.storage.k8s.io/pod.name"]; !ok {
		return nil, errors.New("csi.storage.k8s.io/pod.name key missing in VolumeContext")
	}

	if podLVInfo.UID, ok = req.VolumeContext["csi.storage.k8s.io/pod.uid"]; !ok {
		return nil, errors.New("csi.storage.k8s.io/pod.uid key missing in VolumeContext")
	}

	if podLVInfo.Namespace, ok = req.VolumeContext["csi.storage.k8s.io/pod.namespace"]; !ok {
		return nil, errors.New("csi.storage.k8s.io/pod.namespace key missing in VolumeContext")
	}

	podInfo, err := client.DefaultClient.ClientSet.CoreV1().Pods(podLVInfo.Namespace).Get(context.Background(), podLVInfo.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	podLVInfo.NodeId = podInfo.Spec.NodeName

	return &podLVInfo, nil
}
