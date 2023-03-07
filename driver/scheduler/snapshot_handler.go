package scheduler

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"qiniu.io/rio-csi/logger"
)

func addSnapshot(obj interface{}) {

}

func updateSnapshot(oldObj, newObj interface{}) {
	newSnapshot, ok := SnapshotStructuredObject(newObj)
	if !ok {
		logger.StdLog.Errorf("cant get new Snapshot info %v", newObj)
		return
	}

	switch newSnapshot.Status.State {
	case crd.StatusReady:
		if s, ok := CacheSnapshotMap[newSnapshot.Name]; ok {
			Lock.Lock()
			defer Lock.Unlock()
			s.IsCreated = true
		}
	}

}

func deleteSnapshot(obj interface{}) {

}

// SnapshotStructuredObject get Obj from queue is not readily in lvmnode type. This function would convert obj into lvmnode type.
func SnapshotStructuredObject(obj interface{}) (*apis.Snapshot, bool) {
	unstructuredInterface, ok := obj.(*unstructured.Unstructured)
	if !ok {
		logger.StdLog.Errorf("couldnt type assert obj: %#v to unstructured obj", obj)
		return nil, false
	}
	Snapshot := &apis.Snapshot{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredInterface.UnstructuredContent(), &Snapshot)
	if err != nil {
		logger.StdLog.Errorf("err %s, While converting unstructured obj to typed object\n", err.Error())
		return nil, false
	}
	return Snapshot, true
}
