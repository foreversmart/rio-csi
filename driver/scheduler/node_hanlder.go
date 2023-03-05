package scheduler

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/logger"
)

func addNode(obj interface{}) {

}

func updateNode(oldObj, newObj interface{}) {

}

func deleteNode(obj interface{}) {

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
