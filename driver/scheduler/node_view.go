package scheduler

import (
	"k8s.io/apimachinery/pkg/api/resource"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/utils"
	"regexp"
)

type NodeView struct {
	NodeName            string            `json:"node_name"`
	VolumeNum           int64             `json:"volume_num"`
	SnapshotNum         int64             `json:"snapshot_num"`
	PendingVolumeNum    int64             `json:"pending_num"`
	PendingVolumeSize   int64             `json:"pending_volume_size"` // byte
	PendingSnapshotNum  int64             `json:"pending_snapshot_num"`
	PendingSnapshotSize int64             `json:"pending_snapshot_size"` // byte
	TotalSize           resource.Quantity `json:"total_size"`
	TotalFree           resource.Quantity `json:"total_free"`
	MaxFree             resource.Quantity `json:"max_free"`
	Score               int64             `json:"score"`
}

func NewNodeView(n *apis.RioNode, vgPattern *regexp.Regexp) *NodeView {
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
	return nodeView
}

// ClearCacheData clear node cache data value to zero
func (n *NodeView) ClearCacheData() {
	n.PendingVolumeNum = 0
	n.PendingVolumeSize = 0
	n.PendingSnapshotNum = 0
	n.PendingSnapshotSize = 0
}

// CalcScore calc node score used storage weight is -1 free storage weight 1
// volume Num weight is -1 * 100 Gi, snapshot Num is -1 * 100 Gi
// the more lv num the score is lower
// the more free disk storage the score is higher
// the more usage disk storage the score is lower
func (n *NodeView) CalcScore() {
	used := n.TotalSize
	used.Sub(n.TotalFree)
	free := n.TotalFree
	free.Sub(used)
	score := free.Value() - n.PendingSnapshotSize - n.PendingVolumeSize
	score = score - 100*utils.Gi*n.VolumeNum
	score = score - 100*utils.Gi*n.SnapshotNum
	score = score - 100*utils.Gi*n.PendingVolumeNum
	score = score - 100*utils.Gi*n.PendingSnapshotNum
	n.Score = score
}

// SyncNodeView Sync NodeView cache TODO support more algorithm
func (s *VolumeScheduler) SyncNodeView(nodes []*apis.RioNode) {
	for _, n := range nodes {
		s.NodeViewMap[n.Name] = NewNodeView(n, s.VgPattern)
	}

	return
}
