package scheduler

import (
	"k8s.io/apimachinery/pkg/api/resource"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"strconv"
)

type VolumeView struct {
	Name            string            `json:"name"`
	NodeName        string            `json:"node_name"`
	RequiredStorage resource.Quantity `json:"required_storage"`
	VgName          string            `json:"vg_name"`
	IsCreated       bool              `json:"is_created"`
}

func (s *VolumeScheduler) SyncVolumeView(volumes []*apis.Volume) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	for _, volume := range volumes {
		if volume.Spec.VgPattern != s.VgPatternStr {
			continue
		}

		switch volume.Status.State {
		case crd.StatusPending:
			storageSize, _ := strconv.ParseInt(volume.Spec.Capacity, 10, 64)
			storage := resource.NewQuantity(storageSize, resource.BinarySI)
			s.CacheVolumeMap[volume.Name] = &VolumeView{
				Name:            volume.Name,
				NodeName:        volume.Spec.OwnerNodeID,
				RequiredStorage: *storage,
				VgName:          volume.Spec.VolGroup,
				IsCreated:       false,
			}

		}
	}
}
