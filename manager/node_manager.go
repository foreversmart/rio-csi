package manager

import (
	"context"
	"fmt"
	nodev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/dynamiclister"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/iscsi"
	"qiniu.io/rio-csi/logger"
	"qiniu.io/rio-csi/lvm"
	"qiniu.io/rio-csi/lvm/common/errors"
	"reflect"
	"time"
)

type NodeManager struct {
	NodeID         string
	Namespace      string
	Lister         dynamiclister.Lister
	OwnerReference metav1.OwnerReference
	NodeIP         string
	SyncInterval   time.Duration
}

var nodeResource = schema.GroupVersionResource{
	Group:    apis.SchemeGroupVersion.Group,
	Version:  apis.SchemeGroupVersion.Version,
	Resource: "rionodes",
}

func NewNodeManager(nodeID, namespace string, stopCh chan struct{}) (m *NodeManager, err error) {
	if nodeID == "" || namespace == "" {
		logger.StdLog.Errorf("node ID :%s or namespace :%s is empty", nodeID, namespace)
		return nil, errors.New("node ID or namespace cant be empty")
	}

	k8sNode, err := client.DefaultClient.ClientSet.CoreV1().Nodes().Get(context.TODO(), nodeID, metav1.GetOptions{})
	if err != nil {
		logger.StdLog.Error(err)
		return nil, errors.Wrapf(err, "fetch k8s node %s", lvm.NodeID)
	}

	nodeIP := ""
	for _, address := range k8sNode.Status.Addresses {
		// TODO more robust
		if address.Type == nodev1.NodeInternalIP {
			nodeIP = address.Address
		}
	}

	if nodeIP == "" {
		logger.StdLog.Errorf("cant fetch k8s node %s internal ip", nodeID)
		return nil, errors.New("cant fetch k8s node internal ip")
	}

	// default k8s node gvk
	nodeGVK := &schema.GroupVersionKind{
		Group: "", Version: "v1", Kind: "Node",
	}

	isTrue := true
	ownerRef := metav1.OwnerReference{
		APIVersion: nodeGVK.GroupVersion().String(),
		Kind:       nodeGVK.Kind,
		Name:       k8sNode.Name,
		UID:        k8sNode.GetUID(),
		Controller: &isTrue,
	}

	nodeInformerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client.DefaultClient.DynamicClient, 5*time.Minute,
		namespace, func(options *metav1.ListOptions) {
			options.FieldSelector = fields.OneTermEqualSelector("metadata.name", nodeID).String()
		})

	nodeInformer := nodeInformerFactory.ForResource(nodeResource).Informer()
	lister := dynamiclister.New(nodeInformer.GetIndexer(), nodeResource)
	nodeInformerFactory.Start(stopCh)
	return &NodeManager{
		NodeID:         nodeID,
		Namespace:      namespace,
		Lister:         lister,
		OwnerReference: ownerRef,
		NodeIP:         nodeIP,
		SyncInterval:   time.Second * 60,
	}, nil
}

func (m *NodeManager) Start() {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		<-timer.C
		err := m.Sync()
		if err != nil {
			logger.StdLog.Error(err)
		}
		timer.Reset(m.SyncInterval)
	}
}

func (m *NodeManager) Sync() error {
	cacheNode, err := m.Lister.Namespace(m.Namespace).Get(m.NodeID)
	if err != nil && !k8serror.IsNotFound(err) {
		logger.StdLog.Error(err)
		return err
	}

	m.Lister.Namespace(m.Namespace).List(labels.NewSelector())

	var node *apis.RioNode
	if cacheNode != nil {
		nodeStruct, ok := getNodeStructuredObject(cacheNode)
		if !ok {
			return errors.Errorf("couldn't get node object %#v", cacheNode)
		}

		node = nodeStruct.DeepCopy()
	}

	vgs, err := lvm.ListVolumeGroup(true)
	if err != nil {
		logger.StdLog.Error(err)
		return err
	}

	initiatorName, err := iscsi.GetInitiatorName()
	if err != nil {
		logger.StdLog.Error("GetInitiatorName", err)
		return err
	}

	// if it doesn't exists, create node object
	if node == nil {
		node = &apis.RioNode{
			ObjectMeta: metav1.ObjectMeta{
				Name:            m.NodeID,
				Namespace:       m.Namespace,
				OwnerReferences: []metav1.OwnerReference{m.OwnerReference},
			},
			VolumeGroups: vgs,
			ISCSIInfo: apis.ISCSIInfo{
				// TODO support custom define port
				Portal:        fmt.Sprintf("%s:3260", m.NodeIP),
				InitiatorName: initiatorName,
			},
		}

		logger.StdLog.Infof("rio node controller: creating new node object for %+v", node)
		if _, err = client.DefaultClient.InternalClientSet.RioV1().RioNodes(m.Namespace).Create(context.TODO(), node, metav1.CreateOptions{}); err != nil {
			logger.StdLog.Errorf("create rio node %s/%s: %v", m.Namespace, m.NodeID, err)
			return errors.Errorf("create rio node %s/%s: %v", m.Namespace, m.NodeID, err)
		}

		logger.StdLog.Infof("rio node controller: created node object %s/%s", m.Namespace, m.NodeID)
		return nil
	}

	// if node already exists check if we need to update it
	isNeedUpdate := false
	// validate if owner reference updated.
	if ownerRefs, req := m.isOwnerRefsUpdateRequired(node.OwnerReferences); req {
		logger.StdLog.Infof("rio node controller: node owner references updated current=%+v, required=%+v",
			node.OwnerReferences, ownerRefs)
		node.OwnerReferences = ownerRefs
		isNeedUpdate = true
	}

	// validate if node volume groups are upto date.
	if !equality.Semantic.DeepEqual(node.VolumeGroups, vgs) {
		logger.StdLog.Infof("rio node controller: node volume groups updated current=%+v, required=%+v",
			node.VolumeGroups, vgs)
		node.VolumeGroups = vgs
		isNeedUpdate = true
	}

	// validate if node volume groups are upto date.
	if !equality.Semantic.DeepEqual(node.ISCSIInfo.InitiatorName, initiatorName) {
		logger.StdLog.Infof("rio node controller: node initiatorName updated current=%+v, required=%+v",
			node.ISCSIInfo.InitiatorName, initiatorName)
		node.ISCSIInfo.InitiatorName = initiatorName
		isNeedUpdate = true
	}

	if !isNeedUpdate {
		return nil
	}

	logger.StdLog.Infof("rio node controller: updating node object with %+v", node)
	if _, err = client.DefaultClient.InternalClientSet.RioV1().
		RioNodes(m.Namespace).
		Update(context.TODO(), node, metav1.UpdateOptions{}); err != nil {
		return errors.Errorf("update lvm node %s/%s: %v", m.Namespace, m.NodeID, err)
	}

	logger.StdLog.Infof("rio node controller: updated node object %s/%s", m.Namespace, m.NodeID)

	return nil
}

// getNodeStructuredObject Obj from queue is not readily in lvmnode type. This function would convert obj into lvmnode type.
func getNodeStructuredObject(obj interface{}) (*apis.RioNode, bool) {
	unstructuredInterface, ok := obj.(*unstructured.Unstructured)
	if !ok {
		runtime.HandleError(errors.Errorf("couldnt type assert obj: %#v to unstructured obj", obj))
		return nil, false
	}
	node := &apis.RioNode{}
	err := k8sruntime.DefaultUnstructuredConverter.FromUnstructured(unstructuredInterface.UnstructuredContent(), &node)
	if err != nil {
		runtime.HandleError(fmt.Errorf("err %s, While converting unstructured obj to typed object\n", err.Error()))
		return nil, false
	}
	return node, true
}

// isOwnerRefUpdateRequired validates if relevant owner references is being
// set for rio node. If not, it returns the final owner references that needs
// to be set.
func (m *NodeManager) isOwnerRefsUpdateRequired(ownerRefs []metav1.OwnerReference) ([]metav1.OwnerReference, bool) {
	updated := false
	reqOwnerRef := m.OwnerReference
	for idx := range ownerRefs {
		if ownerRefs[idx].UID != reqOwnerRef.UID {
			continue
		}
		// in case owner reference exists, validate
		// if controller field is set correctly or not.
		if !reflect.DeepEqual(ownerRefs[idx].Controller, reqOwnerRef.Controller) {
			updated = true
			ownerRefs[idx].Controller = reqOwnerRef.Controller
		}
		return ownerRefs, updated
	}
	updated = true
	ownerRefs = append(ownerRefs, reqOwnerRef)
	return ownerRefs, updated
}
