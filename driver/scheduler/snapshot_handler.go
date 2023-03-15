package scheduler

import (
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"qiniu.io/rio-csi/logger"
)

func (s *VolumeScheduler) addSnapshot(obj interface{}) {

}

func (s *VolumeScheduler) updateSnapshot(oldObj, newObj interface{}) {
	newSnapshot, ok := SnapshotStructuredObject(newObj)
	if !ok {
		logger.StdLog.Errorf("cant get new Snapshot info %v", newObj)
		return
	}

	switch newSnapshot.Status.State {
	case crd.StatusReady:
		s.Lock.Lock()
		defer s.Lock.Unlock()
		if snap, ok := s.CacheSnapshotMap[newSnapshot.Name]; ok {
			snap.IsCreated = true
		}
	}

}

func (s *VolumeScheduler) deleteSnapshot(obj interface{}) {

}

// SnapshotStructuredObject get Obj from queue is not readily in lvmnode type. This function would convert obj into lvmnode type.
func SnapshotStructuredObject(obj interface{}) (*apis.Snapshot, bool) {
	snap, ok := obj.(*apis.Snapshot)
	if !ok {
		logger.StdLog.Errorf("couldnt type assert obj: %#v to snapshot obj", obj)
		return nil, false
	}

	return snap, true
}
