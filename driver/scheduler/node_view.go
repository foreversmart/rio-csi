package scheduler

import (
	"k8s.io/apimachinery/pkg/api/resource"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"regexp"
)

type NodeView struct {
	NodeName    string            `json:"node_name"`
	VolumeNum   int64             `json:"volume_num"`
	SnapshotNum int64             `json:"snapshot_num"`
	TotalSize   resource.Quantity `json:"total_size"`
	TotalFree   resource.Quantity `json:"total_free"`
	MaxFree     resource.Quantity `json:"max_free"`
}

var (
	NodeViewMap map[string]*NodeView
)

func init() {
	NodeViewMap = make(map[string]*NodeView)
}

// SyncNodeView Sync NodeView cache TODO support more algorithm
func SyncNodeView(nodes []*apis.RioNode, vgPattern *regexp.Regexp) {
	for _, n := range nodes {
		nodeView := &NodeView{
			NodeName: n.Name,
		}

		maxFree := resource.Quantity{}
		for _, vg := range n.VolumeGroups {
			if vgPattern.MatchString(vg.Name) {
				nodeView.VolumeNum = nodeView.VolumeNum + int64(vg.LVCount)
				nodeView.SnapshotNum = nodeView.SnapshotNum + int64(vg.SnapCount)
				nodeView.TotalSize.Add(vg.Size)
				nodeView.TotalFree.Add(vg.Free)
				if maxFree.Cmp(vg.Free) < 0 {
					maxFree = vg.Free
				}
			}
		}
		nodeView.MaxFree = maxFree
		NodeViewMap[n.Name] = nodeView
	}

	return
}
