package scheduler

import (
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"qiniu.io/rio-csi/logger"
)

func (s *VolumeScheduler) addVolume(obj interface{}) {

}

func (s *VolumeScheduler) updateVolume(oldObj, newObj interface{}) {
	newVolume, ok := VolumeStructuredObject(newObj)
	if !ok {
		logger.StdLog.Errorf("cant get new volume info %v", newObj)
		return
	}

	switch newVolume.Status.State {
	case crd.StatusReady, crd.StatusCreated, crd.StatusCloning:
		s.Lock.Lock()
		defer s.Lock.Unlock()
		if volumeView, ok := s.CacheVolumeMap[newVolume.Name]; ok {
			volumeView.IsCreated = true
		}
	case crd.StatusFailed:
		// TODO check error msg to decide whether space should be freed
	}

}

func (s *VolumeScheduler) deleteVolume(obj interface{}) {

}

// VolumeStructuredObject get Obj from queue is not readily in lvmnode type. This function would convert obj into lvmnode type.
func VolumeStructuredObject(obj interface{}) (*apis.Volume, bool) {
	volume, ok := obj.(*apis.Volume)
	if !ok {
		logger.StdLog.Errorf("couldnt type assert obj: %#v to volume obj", obj)
		return nil, false
	}

	return volume, true
}
