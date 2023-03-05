package scheduler

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/logger"
)

func addVolume(obj interface{}) {

}

func updateVolume(oldObj, newObj interface{}) {

}

func deleteVolume(obj interface{}) {

}

// VolumeStructuredObject get Obj from queue is not readily in lvmnode type. This function would convert obj into lvmnode type.
func VolumeStructuredObject(obj interface{}) (*apis.Volume, bool) {
	unstructuredInterface, ok := obj.(*unstructured.Unstructured)
	if !ok {
		logger.StdLog.Errorf("couldnt type assert obj: %#v to unstructured obj", obj)
		return nil, false
	}
	volume := &apis.Volume{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredInterface.UnstructuredContent(), &volume)
	if err != nil {
		logger.StdLog.Errorf("err %s, While converting unstructured obj to typed object\n", err.Error())
		return nil, false
	}
	return volume, true
}
