package scheduler

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

type VolumeView struct {
	Name            string            `json:"name"`
	NodeName        string            `json:"node_name"`
	RequiredStorage resource.Quantity `json:"required_storage"`
}

var (
	// CacheVolumeMap to record volume which didn't success to create
	CacheVolumeMap map[string]*VolumeView
)
