package scheduler

import (
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/cache"
	"qiniu.io/rio-csi/client"
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
	client.DefaultInformer.Rio().V1().RioNodes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    addNode,
		UpdateFunc: updateNode,
		DeleteFunc: deleteNode,
	})

	client.DefaultInformer.Rio().V1().Volumes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    addVolume,
		UpdateFunc: updateVolume,
		DeleteFunc: deleteVolume,
	})

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

	Lock.Lock()
	defer Lock.Unlock()

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

// NodeSort calc node score and sort at desc
func (s *VolumeScheduler) NodeSort(req *csi.CreateVolumeRequest) (nodes []*NodeView) {
	// clear caching data
	for _, node := range NodeViewMap {
		node.ClearCacheData()
	}

	// recalculate caching data
	for _, v := range CacheVolumeMap {
		NodeViewMap[v.NodeName].PendingVolumeNum = NodeViewMap[v.NodeName].PendingVolumeNum + 1
		NodeViewMap[v.NodeName].PendingVolumeSize = NodeViewMap[v.NodeName].PendingVolumeSize + v.RequiredStorage.Value()
	}

	for _, v := range CacheSnapshotMap {
		NodeViewMap[v.NodeName].PendingSnapshotNum = NodeViewMap[v.NodeName].PendingSnapshotNum + 1
		NodeViewMap[v.NodeName].PendingSnapshotSize = NodeViewMap[v.NodeName].PendingSnapshotSize + v.RequiredStorage.Value()
	}

	// recalculate node view score
	nodes = make([]*NodeView, len(NodeViewMap))
	for _, node := range NodeViewMap {
		node.CalcScore()
		// deep copy to result
		nodes = append(nodes, &NodeView{
			NodeName:            node.NodeName,
			VolumeNum:           node.VolumeNum,
			SnapshotNum:         node.SnapshotNum,
			PendingVolumeNum:    node.PendingVolumeNum,
			PendingVolumeSize:   node.PendingVolumeSize,
			PendingSnapshotNum:  node.PendingSnapshotNum,
			PendingSnapshotSize: node.PendingSnapshotSize,
			TotalSize:           node.TotalSize,
			TotalFree:           node.TotalFree,
			MaxFree:             node.MaxFree,
			Score:               node.Score,
		})
	}

	//

	// sort the filtered node map
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Score > nodes[j].Score
	})

	return nodes
}
