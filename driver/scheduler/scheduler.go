package scheduler

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/dynamiclister"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/driver/dparams"
	"sync"
	"time"
)

var (
	Lock sync.Mutex
)

type VolumeScheduler struct {
}

func NewVolumeScheduler() {
	nodeInformerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client.DefaultClient.DynamicClient, 5*time.Minute,
		namespace, func(options *metav1.ListOptions) {
			options.FieldSelector = fields.OneTermEqualSelector("metadata.name", nodeID).String()
		})

	nodeInformer := nodeInformerFactory.ForResource(nodeResource).Informer()
	lister := dynamiclister.New(nodeInformer.GetIndexer(), nodeResource)
}

// GetNode BalancedResourceAllocation
func GetNode(param *dparams.VolumeParams) (node *apis.RioNode, err error) {

	return
}
