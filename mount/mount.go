package mount

import apis "qiniu.io/rio-csi/api/rio/v1"

type MountData struct {
}

func (m *MountData) Mount(vol *apis.Volume, mountinfo *Info, podLVInfo *PodInfo) error {

}
