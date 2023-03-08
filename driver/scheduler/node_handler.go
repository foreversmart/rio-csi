package scheduler

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/logger"
	"regexp"
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
	var vgPattern *regexp.Regexp
	// update node view
	s.NodeViewMap[newNode.Name] = NewNodeView(newNode, vgPattern)
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
	unstructuredInterface, ok := obj.(*unstructured.Unstructured)
	if !ok {
		logger.StdLog.Errorf("couldnt type assert obj: %#v to unstructured obj", obj)
		return nil, false
	}
	node := &apis.RioNode{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredInterface.UnstructuredContent(), &node)
	if err != nil {
		logger.StdLog.Errorf("err %s, While converting unstructured obj to typed object\n", err.Error())
		return nil, false
	}
	return node, true
}
