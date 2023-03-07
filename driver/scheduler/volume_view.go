package scheduler

import (
	"k8s.io/apimachinery/pkg/api/resource"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"strconv"
)

var (
	// CacheVolumeMap to record volume which didn't success to create
	CacheVolumeMap map[string]*VolumeView
)

type VolumeView struct {
	Name            string            `json:"name"`
	NodeName        string            `json:"node_name"`
	RequiredStorage resource.Quantity `json:"required_storage"`
	VgName          string            `json:"vg_name"`
	IsCreated       bool              `json:"is_created"`
}

func SyncVolumeView(volumes []*apis.Volume) {
	Lock.Lock()
	defer Lock.Unlock()
	for _, volume := range volumes {
		switch volume.Status.State {
		case crd.StatusPending:
			storageSize, _ := strconv.ParseInt(volume.Spec.Capacity, 10, 64)
			storage := resource.NewQuantity(storageSize, resource.BinarySI)
			CacheVolumeMap[volume.Name] = &VolumeView{
				Name:            volume.Name,
				NodeName:        volume.Spec.OwnerNodeID,
				RequiredStorage: *storage,
				VgName:          volume.Spec.VolGroup,
				IsCreated:       false,
			}

		}
	}
}
