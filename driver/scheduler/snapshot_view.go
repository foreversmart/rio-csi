package scheduler

import (
	"k8s.io/apimachinery/pkg/api/resource"
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
}
