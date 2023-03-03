package scheduler

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/logger"
)

func filterTopologyRequirement(topologyReq *csi.TopologyRequirement) ([]string, error) {
	if topologyReq == nil {
		return nil, nil
	}

	topo := topologyReq.Preferred
	if len(topo) == 0 {
		// if preferred list is empty, use the requisite
		logger.StdLog.Error("TopologyRequirement topology is not provided")
		topo = topologyReq.Requisite
	}

	if len(topo) == 0 {
		logger.StdLog.Error("topology information is not provided")
		return nil, nil
	}

	return filterTopology(topo)
}

// filterTopology gets the node list which satisfies the topology info
func filterTopology(topo []*csi.Topology) ([]string, error) {
	var nodeList []string

	list, err := client.DefaultClient.ClientSet.CoreV1().Nodes().List(nil, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, node := range list.Items {
		for _, prf := range topo {
			nodeFiltered := false
			for key, value := range prf.Segments {
				if node.Labels[key] != value {
					nodeFiltered = true
					break
				}
			}
			if !nodeFiltered {
				nodeList = append(nodeList, node.Name)
				break
			}
		}
	}

	return nodeList, nil
}
