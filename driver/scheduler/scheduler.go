package scheduler

import (
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/cache"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/logger"
	"regexp"
	"sort"
	"sync"
)

type VolumeScheduler struct {
	VgPatternStr string
	VgPattern    *regexp.Regexp
	NodeViewMap  map[string]*NodeView
	// CacheVolumeMap to record volume which didn't success to create
	CacheVolumeMap map[string]*VolumeView
	// CacheSnapshotMap to record Snapshot which didn't success to create
	CacheSnapshotMap map[string]*SnapshotView
	Lock             sync.Mutex
}

func NewVolumeScheduler(vgPatternStr string) (s *VolumeScheduler, err error) {
	s = &VolumeScheduler{
		VgPatternStr:     vgPatternStr,
		NodeViewMap:      make(map[string]*NodeView),
		CacheVolumeMap:   make(map[string]*VolumeView),
		CacheSnapshotMap: make(map[string]*SnapshotView),
	}

	err = s.Sync()

	if s.VgPattern, err = regexp.Compile(s.VgPatternStr); err != nil {
		return nil, fmt.Errorf("invalid vgpattern format  %v: %v", s.VgPatternStr, err)
	}

	client.DefaultInformer.Rio().V1().RioNodes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.addNode,
		UpdateFunc: s.updateNode,
		DeleteFunc: s.deleteNode,
	})

	client.DefaultInformer.Rio().V1().Volumes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.addVolume,
		UpdateFunc: s.updateVolume,
		DeleteFunc: s.deleteVolume,
	})

	client.DefaultInformer.Rio().V1().Snapshots().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.addSnapshot,
		UpdateFunc: s.updateSnapshot,
		DeleteFunc: s.deleteSnapshot,
	})

	return
}

func (s *VolumeScheduler) Sync() error {
	nodes, err := client.DefaultInformer.Rio().V1().RioNodes().Lister().List(nil)
	if err != nil {
		logger.StdLog.Errorf("list node error", err)
		return err
	}

	s.SyncNodeView(nodes)

	volumes, err := client.DefaultInformer.Rio().V1().Volumes().Lister().List(nil)
	if err != nil {
		logger.StdLog.Errorf("list node error", err)
		return err
	}

	s.SyncVolumeView(volumes)

	snapshots, err := client.DefaultInformer.Rio().V1().Snapshots().Lister().List(nil)
	if err != nil {
		logger.StdLog.Errorf("list node error", err)
		return err
	}

	s.SyncSnapshotView(snapshots)
	return nil

}

// ScheduleVolume volume to a specific node
// TODO support multi Vg allocate
func (s *VolumeScheduler) ScheduleVolume(req *csi.CreateVolumeRequest) (nodeName string, err error) {
	filterNodesMap, err := filterTopologyRequirement(req.AccessibilityRequirements)
	if err != nil {
		logger.StdLog.Errorf("filterTopologyRequirement %v", err)
		return
	}

	s.Lock.Lock()
	defer s.Lock.Unlock()

	sortNodes := s.NodeSort(req)
	for _, node := range sortNodes {

		if _, ok := filterNodesMap[node.NodeName]; !ok {
			continue
		}

		requiredStorage := resource.NewQuantity(req.CapacityRange.RequiredBytes, resource.BinarySI)
		if node.MaxFree.Cmp(*requiredStorage) > 0 {
			// cache pending volume data
			s.CacheVolumeMap[req.Name] = &VolumeView{
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
	for _, node := range s.NodeViewMap {
		node.ClearCacheData()
	}

	// recalculate caching data
	for _, v := range s.CacheVolumeMap {
		s.NodeViewMap[v.NodeName].PendingVolumeNum = s.NodeViewMap[v.NodeName].PendingVolumeNum + 1
		s.NodeViewMap[v.NodeName].PendingVolumeSize = s.NodeViewMap[v.NodeName].PendingVolumeSize + v.RequiredStorage.Value()
	}

	for _, v := range s.CacheSnapshotMap {
		s.NodeViewMap[v.NodeName].PendingSnapshotNum = s.NodeViewMap[v.NodeName].PendingSnapshotNum + 1
		s.NodeViewMap[v.NodeName].PendingSnapshotSize = s.NodeViewMap[v.NodeName].PendingSnapshotSize + v.RequiredStorage.Value()
	}

	// recalculate node view score
	nodes = make([]*NodeView, len(s.NodeViewMap))
	for _, node := range s.NodeViewMap {
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

	// sort the filtered node map
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Score > nodes[j].Score
	})

	return nodes
}
