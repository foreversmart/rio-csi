package scheduler

import (
	"k8s.io/apimachinery/pkg/api/resource"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"strconv"
)

type SnapshotView struct {
	Name            string            `json:"name"`
	NodeName        string            `json:"node_name"`
	RequiredStorage resource.Quantity `json:"required_storage"`
	VgName          string            `json:"vg_name"`
	IsCreated       bool              `json:"is_created"`
}

func (s *VolumeScheduler) SyncSnapshotView(snapshots []*apis.Snapshot) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	for _, snap := range snapshots {
		if (snap.Spec.VgPattern != "" && snap.Spec.VgPattern == s.VgPatternStr) || s.VgPattern.MatchString(snap.Spec.VolGroup) {
			switch snap.Status.State {
			case crd.StatusPending:
				storageSize, _ := strconv.ParseInt(snap.Spec.SnapSize, 10, 64)
				storage := resource.NewQuantity(storageSize, resource.BinarySI)
				s.CacheSnapshotMap[snap.Name] = &SnapshotView{
					Name:            snap.Name,
					NodeName:        snap.Spec.OwnerNodeID,
					RequiredStorage: *storage,
					VgName:          snap.Spec.VolGroup,
					IsCreated:       false,
				}
			}
		}
	}
}
