package mount

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/exec"
	"k8s.io/utils/mount"
	"os"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/iscsi"
	"qiniu.io/rio-csi/logger"
)

func iscsiMount(vol *apis.Volume, mountInfo *Info, podLVInfo *PodInfo) error {
	node, getErr := client.DefaultClient.InternalClientSet.RioV1().RioNodes(vol.Namespace).Get(context.TODO(), vol.Spec.OwnerNodeID, metav1.GetOptions{})
	if getErr != nil {
		logger.StdLog.Errorf("get %s rio node %s info error %v", vol.Namespace, vol.Spec.OwnerNodeID, err)
		return getErr
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
		logger.StdLog.Error(connectErr)
		return connectErr
	}

	if devicePath == "" {
		logger.StdLog.Error("connect reported success, but no path returned")
		return fmt.Errorf("connect reported success, but no path returned")
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
}
