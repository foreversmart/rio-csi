package mount

import (
	"fmt"
	"k8s.io/utils/exec"
	"k8s.io/utils/mount"
	"os"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/logger"
)

func iscsiMount(vol *apis.Volume, mountInfo *Info, podLVInfo *PodInfo) error {
	devicePath := mountInfo.DevicePath
	mntPath := mountInfo.MountPath
	mounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: exec.New()}

	// check mount point
	notMnt, mountErr := mounter.IsLikelyNotMountPoint(mntPath)
	if mountErr != nil && !os.IsNotExist(mountErr) {
		logger.StdLog.Errorf("heuristic determination of mount point failed %s error %v", mntPath, mountErr)
		return fmt.Errorf("heuristic determination of mount point failed:%v", mountErr)
	}

	if !notMnt {
		logger.StdLog.Infof("iscsi: %s already mounted", mntPath)
		return nil
	}

	if err := os.MkdirAll(mntPath, 0o750); err != nil {
		logger.StdLog.Errorf("iscsi: failed to mkdir %s, error", mntPath)
		return err
	}

	err := mounter.FormatAndMount(devicePath, mntPath, mountInfo.FSType, mountInfo.MountOptions)
	if err != nil {
		logger.StdLog.Errorf("iscsi: failed to mount iscsi volume %s [%s] to %s, error %v", devicePath, mountInfo.FSType, mntPath, err)
	}

	return nil
}
