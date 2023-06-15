package mount

import (
	utilexec "k8s.io/utils/exec"
	"k8s.io/utils/mount"
	"os"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/lib/mount/mtypes"
	"qiniu.io/rio-csi/logger"
	"strings"
)

// lvm related constants
const (
	DevPath       = "/dev/"
	DevMapperPath = "/dev/mapper/"
	// MinExtentRoundOffSize represents minimum size (256Mi) to roundoff the volume
	// group size in case of thin pool provisioning
	MinExtentRoundOffSize = 268435456

	// BlockCleanerCommand is the command used to clean filesystem on the device
	BlockCleanerCommand = "wipefs"
)

// GetVolumeDevicePath returns device path for the given volume
func GetVolumeDevicePath(vol *apis.Volume) string {
	volume := vol.Spec.VolGroup + "/" + vol.Name
	devicePath := DevPath + volume

	return devicePath
}

// GetLVMVolumeDevPath returns devpath for the given volume
func GetLVMVolumeDevPath(vol *apis.Volume) (string, error) {
	// LVM doubles the hiphen for the mapper device name
	// and uses single hiphen to separate volume group from volume
	vg := strings.Replace(vol.Spec.VolGroup, "-", "--", -1)

	lv := strings.Replace(vol.Name, "-", "--", -1)
	dev := DevMapperPath + vg + "-" + lv

	return dev, nil
}

// GetMounts gets mountpoints for the specified volume
func GetMounts(dev string) ([]string, error) {

	var (
		currentMounts []string
		err           error
		mountList     []mount.MountPoint
	)

	mounter := mount.New("")
	// Get list of mounted paths present with the node
	if mountList, err = mounter.List(); err != nil {
		return nil, err
	}
	for _, mntInfo := range mountList {
		if mntInfo.Device == dev {
			currentMounts = append(currentMounts, mntInfo.Path)
		}
	}
	return currentMounts, nil
}

// IsMountPath returns true if path is a mount path
func IsMountPath(path string) bool {

	var (
		err       error
		mountList []mount.MountPoint
	)

	mounter := mount.New("")
	// Get list of mounted paths present with the node
	if mountList, err = mounter.List(); err != nil {
		return false
	}
	for _, mntInfo := range mountList {
		if mntInfo.Path == path {
			return true
		}
	}
	return false
}

// FormatAndMountVol formats and mounts the created volume to the desired mount path
func FormatAndMountVol(devicePath string, mountInfo *mtypes.VolumeInfo) error {
	mounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: utilexec.New()}

	err := mounter.FormatAndMount(devicePath, mountInfo.MountPath, mountInfo.FSType, mountInfo.MountOptions)
	if err != nil {
		logger.StdLog.Errorf(
			"lvm: failed to mount volume %s [%s] to %s, error %v",
			devicePath, mountInfo.FSType, mountInfo.MountPath, err,
		)
		return err
	}

	return nil
}

func makeFile(pathname string) error {
	f, err := os.OpenFile(pathname, os.O_CREATE, os.FileMode(0644))
	defer func(f *os.File) {
		err = f.Close()
		logger.StdLog.Errorf("failed to close file %s error: %v", f.Name(), err)
	}(f)
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}
