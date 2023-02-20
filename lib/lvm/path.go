package lvm

import (
	"os"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"strings"
)

const (
	DevPath       = "/dev/"
	DevMapperPath = "/dev/mapper/"
)

// GetVolumeDevMapperPath returns dev mapper path for the given volume
func GetVolumeDevMapperPath(vol *apis.Volume) string {
	// LVM doubles the hiphen for the mapper device name
	// and uses single hiphen to separate volume group from volume
	vg := strings.Replace(vol.Spec.VolGroup, "-", "--", -1)

	lv := strings.Replace(vol.Name, "-", "--", -1)
	dev := DevMapperPath + vg + "-" + lv

	return dev
}

// GetVolumeDevPath returns dev path for the given volume
func GetVolumeDevPath(vol *apis.Volume) string {
	dev := GetDevPath(vol.Spec.VolGroup, vol.Name)
	return dev
}

// GetDevPath returns dev path for the given vg name and volume name
func GetDevPath(vgName, volName string) string {
	dev := DevPath + vgName + "/" + volName
	return dev
}

// CheckPathExist check the given path exist in os
func CheckPathExist(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
