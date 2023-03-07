package scheduler

import (
	"k8s.io/apimachinery/pkg/api/resource"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"strconv"
)

var (
	// CacheSnapshotMap to record Snapshot which didn't success to create
	CacheSnapshotMap map[string]*SnapshotView
)

type SnapshotView struct {
	Name            string            `json:"name"`
	NodeName        string            `json:"node_name"`
	RequiredStorage resource.Quantity `json:"required_storage"`
	VgName          string            `json:"vg_name"`
	IsCreated       bool              `json:"is_created"`
}

func SyncSnapshotView(snapshots []*apis.Snapshot) {
	Lock.Lock()
	defer Lock.Unlock()
	for _, s := range snapshots {
		switch s.Status.State {
		case crd.StatusPending:
			storageSize, _ := strconv.ParseInt(s.Spec.SnapSize, 10, 64)
			storage := resource.NewQuantity(storageSize, resource.BinarySI)
			CacheSnapshotMap[s.Name] = &SnapshotView{
				Name:            s.Name,
				NodeName:        s.Spec.OwnerNodeID,
				RequiredStorage: *storage,
				VgName:          s.Spec.VolGroup,
				IsCreated:       false,
			}

		}
	}
}
