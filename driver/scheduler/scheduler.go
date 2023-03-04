package scheduler

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/driver/dparams"
	"qiniu.io/rio-csi/logger"
	"sort"
	"sync"
)

var (
	Lock sync.Mutex
)

type VolumeScheduler struct {
}

func NewVolumeScheduler() {

}

// GetNode BalancedResourceAllocation
func (s *VolumeScheduler) GetNode(req *csi.CreateVolumeRequest, param *dparams.VolumeParams) (node *apis.RioNode, err error) {
	nodes, err := client.DefaultInformer.Rio().V1().RioNodes().Lister().List(nil)
	if err != nil {
		logger.StdLog.Errorf("list node error", err)
		return nil, err
	}

	return
}

func (s *VolumeScheduler) Runner() {
	nodes, err := client.DefaultInformer.Rio().V1().RioNodes().Informer().AddEventHandler()
}

func (s *VolumeScheduler) Schedule(req *csi.CreateVolumeRequest, param *dparams.VolumeParams) {
	nodes, err := client.DefaultInformer.Rio().V1().RioNodes().Lister().List(nil)
	if err != nil {
		logger.StdLog.Errorf("list node error", err)
		return nil, err
	}

	filterNodes, err := filterTopologyRequirement(req.AccessibilityRequirements)
	if err != nil {
		logger.StdLog.Errorf("filterTopologyRequirement %v", err)
		return
	}

}

func (s *VolumeScheduler) NodeSort(req *csi.CreateVolumeRequest) (nodes []*NodeView) {
	nodes = make([]*NodeView, len(NodeViewMap))
	for _, node := range NodeViewMap {
		node.CalcScore()
		nodes = append(nodes, node)
	}

	//

	// sort the filtered node map
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Score() < fmap[j].Value
	})
}
