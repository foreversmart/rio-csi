package lvm

import (
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
	dev := DevPath + vol.Spec.VolGroup + "/" + vol.Name
	return dev
}
