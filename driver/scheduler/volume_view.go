package scheduler

import (
	"k8s.io/apimachinery/pkg/api/resource"
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
