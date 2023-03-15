package scheduler

import (
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/logger"
)

func (s *VolumeScheduler) addNode(obj interface{}) {

}

func (s *VolumeScheduler) updateNode(oldObj, newObj interface{}) {
	newNode, ok := NodeStructuredObject(newObj)
	if !ok {
		logger.StdLog.Errorf("cant get new node info %v", newObj)
		return
	}

	s.Lock.Lock()
	defer s.Lock.Unlock()
	// update node view
	s.NodeViewMap[newNode.Name] = NewNodeView(newNode, s.VgPattern)
	// free finish volume space cache
	for name, volume := range s.CacheVolumeMap {
		if volume.NodeName == newNode.Name && volume.IsCreated {
			delete(s.CacheVolumeMap, name)
		}
	}

	// free finish snapshot space cache
	for name, snapshot := range s.CacheSnapshotMap {
		if snapshot.NodeName == newNode.Name && snapshot.IsCreated {
			delete(s.CacheSnapshotMap, name)
		}
	}

}

func (s *VolumeScheduler) deleteNode(obj interface{}) {

}

// NodeStructuredObject get Obj from queue is not readily in lvmnode type. This function would convert obj into lvmnode type.
func NodeStructuredObject(obj interface{}) (*apis.RioNode, bool) {
	node, ok := obj.(*apis.RioNode)
	if !ok {
		logger.StdLog.Errorf("couldnt type assert obj: %#v to rionode obj", obj)
		return nil, false
	}

	return node, true
}
