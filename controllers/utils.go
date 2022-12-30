package controllers

import (
	"errors"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/iscsi"
	"qiniu.io/rio-csi/logger"
)

// CreateTargetAcl check target whether exist, if not create
// TODO add initiator acl rules partial not all
func CreateTargetAcl(namespace, target, username, password string) (err error) {
	nodes, listErr := client.DefaultClient.InternalClientSet.RioV1().RioNodes(namespace).List(context.TODO(), metav1.ListOptions{})
	if listErr != nil {
		logger.StdLog.Errorf("list %s rio node info error %v", namespace, err)
		return listErr
	}

	currentAclList, err := iscsi.ListTargetAcl(target)
	if err != nil {
		logger.StdLog.Errorf("ListTargetAcl %s error %v", target, err)
		return err
	}

	aclMap := make(map[string]bool)
	for _, v := range currentAclList {
		aclMap[v] = true
	}

	count := 0
	for _, node := range nodes.Items {
		if aclMap[node.ISCSIInfo.InitiatorName] {
			count++
			continue
		}

		// not exist create
		_, err = iscsi.SetUpTargetAcl(target, node.ISCSIInfo.InitiatorName, username, password)
		if err != nil {
			logger.StdLog.Errorf("SetUpTargetAcl target %s initiator %s error %v", target, node.ISCSIInfo.InitiatorName, err)
			continue
		}

		// success
		count++
	}

	// if all initiator is set dot check anymore
	if count == len(nodes.Items) {
		return nil
	}

	return errors.New("acl not set all for the nodes")
}
