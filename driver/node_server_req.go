package driver

import (
	"errors"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/lvm"
	"strings"
)

func (ns *nodeServer) validateNodePublishReq(req *csi.NodePublishVolumeRequest) error {
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

func (ns *nodeServer) validateNodeUnPublishReq(req *csi.NodeUnpublishVolumeRequest) error {
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
func GetVolAndMountInfo(req *csi.NodePublishVolumeRequest) (*apis.Volume, *lvm.MountInfo, error) {
	var mountinfo lvm.MountInfo

	mountinfo.FSType = req.GetVolumeCapability().GetMount().GetFsType()
	mountinfo.MountPath = req.GetTargetPath()
	mountinfo.MountOptions = append(mountinfo.MountOptions, req.GetVolumeCapability().GetMount().GetMountFlags()...)

	if req.GetReadonly() {
		mountinfo.MountOptions = append(mountinfo.MountOptions, "ro")
	} else {
		mountinfo.MountOptions = append(mountinfo.MountOptions, "rw")
	}

	volName := strings.ToLower(req.GetVolumeId())

	vol, err := lvm.GetVolume(volName)
	if err != nil {
		return nil, nil, err
	}

	return vol, &mountinfo, nil
}

func getPodLVInfo(req *csi.NodePublishVolumeRequest) (*lvm.PodLVInfo, error) {
	var podLVInfo lvm.PodLVInfo
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

	if podLVInfo.LVGroup, ok = req.VolumeContext["openebs.io/volgroup"]; !ok {
		return nil, errors.New("openebs.io/volgroup key missing in VolumeContext")
	}

	podInfo, err := client.DefaultClient.ClientSet.CoreV1().Pods(podLVInfo.Namespace).Get(nil, podLVInfo.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	podLVInfo.NodeId = podInfo.Spec.NodeName

	return &podLVInfo, nil
}
