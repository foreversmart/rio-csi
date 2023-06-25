package mount

import (
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/lib/iscsi"
	"qiniu.io/rio-csi/logger"
)

// NewIscsiConnector help to use vol to create volume self connector
func NewIscsiConnector(vol *apis.Volume, iscsiUsername, iscsiPassword string) (connector *iscsi.Connector, err error) {
	node, err := client.DefaultClient.InternalClientSet.RioV1().RioNodes(vol.Namespace).Get(context.TODO(), vol.Spec.OwnerNodeID, metav1.GetOptions{})
	if err != nil {
		logger.StdLog.Errorf("get %s rio node %s info error %v", vol.Namespace, vol.Spec.OwnerNodeID, err)
		return nil, err
	}
	// mount on different nodes using iscsi
	connector = &iscsi.Connector{
		AuthType:      "chap",
		VolumeName:    vol.Name,
		TargetIqn:     vol.Spec.IscsiTarget,
		TargetPortals: []string{node.ISCSIInfo.Portal},
		Lun:           vol.Spec.IscsiLun,
		DiscoverySecrets: iscsi.Secrets{
			SecretsType: "chap",
			UserName:    iscsiUsername,
			Password:    iscsiPassword,
		},
		DoDiscovery:     true,
		DoCHAPDiscovery: true,
	}

	return
}
