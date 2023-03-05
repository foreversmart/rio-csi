package scheduler

import (
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/apimachinery/pkg/api/resource"
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

// // GetNode BalancedResourceAllocation
//
//	func (s *VolumeScheduler) GetNode(req *csi.CreateVolumeRequest, param *dparams.VolumeParams) (node *apis.RioNode, err error) {
//		nodes, err := client.DefaultInformer.Rio().V1().RioNodes().Lister().List(nil)
//		if err != nil {
//			logger.StdLog.Errorf("list node error", err)
//			return nil, err
//		}
//
//		return
//	}
//
// ScheduleVolume volume to a specific node
// TODO support multi Vg allocate
func (s *VolumeScheduler) ScheduleVolume(req *csi.CreateVolumeRequest) (nodeName string, err error) {
	//nodes, err := client.DefaultInformer.Rio().V1().RioNodes().Lister().List(nil)
	//if err != nil {
	//	logger.StdLog.Errorf("list node error", err)
	//	return nil, err
	//}

	filterNodesMap, err := filterTopologyRequirement(req.AccessibilityRequirements)
	if err != nil {
		logger.StdLog.Errorf("filterTopologyRequirement %v", err)
		return
	}

	sortNodes := s.NodeSort(req)
	for _, node := range sortNodes {

		if _, ok := filterNodesMap[node.NodeName]; !ok {
			continue
		}

		requiredStorage := resource.NewQuantity(req.CapacityRange.RequiredBytes, resource.BinarySI)
		if node.MaxFree.Cmp(*requiredStorage) > 0 {
			// cache pending volume data
			CacheVolumeMap[req.Name] = &VolumeView{
				Name:            req.Name,
				NodeName:        node.NodeName,
				RequiredStorage: *requiredStorage,
			}
			return node.NodeName, nil
		}
	}

	return "", fmt.Errorf("cant find a suitable node")
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
		return nodes[i].Score > nodes[j].Score
	})

	return nodes
}
