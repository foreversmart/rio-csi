package controllers

import (
	"fmt"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/crd"
	"qiniu.io/rio-csi/lib/iscsi"
	"qiniu.io/rio-csi/lib/mount"
	"qiniu.io/rio-csi/lib/mount/mtypes"
	"qiniu.io/rio-csi/logger"
)

// CheckAndRecoveryDisk check all disk from csi cr disk and recovery disk status
func CheckAndRecoveryDisk(nodeID, iscsiUsername, iscsiPassword string) {
	targets, err := iscsi.ListTarget()
	if err != nil {
		logger.StdLog.Error("List target error", err)
		return
	}

	targetsMap := make(map[string]bool)
	for _, v := range targets {
		targetsMap[v] = true
	}

	logger.StdLog.Info("Check Disk IScsi start")

	// fetch iscsi current sessions
	sessions, err := iscsi.GetCurrentSessions()
	if err != nil {
		logger.StdLog.Errorf("iscsi get current sessions node %s error %v", nodeID, err)
		return
	}

	sessionMap := make(map[string]bool)
	for _, session := range sessions {
		sessionMap[session.IQN] = true
	}

	skip := ""
	limit := int64(100)
	for {
		resp, conStr, err := crd.ListVolumes(skip, limit)
		if err != nil {
			logger.StdLog.Errorf("ListVolumes skip %d limit %d error %v", skip, limit, err)
			return
		}

		for _, vol := range resp {
			if nodeID == vol.Spec.OwnerNodeID {
				CheckAndRecoveryDiskIscsi(vol, iscsiUsername, iscsiPassword, targetsMap)
			}

			for _, no := range vol.Spec.MountNodes {
				if no.PodInfo.NodeId == nodeID {
					// volume session not exist on this node do recovery
					if _, ok := sessionMap[vol.Spec.IscsiTarget]; !ok {
						RecoveryDiskIscsiSession(vol, no, iscsiUsername, iscsiPassword)
					}

					// recovery once then exist because one vol may mount many pod on the same node
					break
				}
			}
		}

		if conStr == "" {
			break
		}

		skip = conStr
	}
	logger.StdLog.Info("Check Disk IScsi Finish")
}

// RecoveryDiskIscsiSession recovery iscsi session
func RecoveryDiskIscsiSession(vol apis.Volume, info *mtypes.Info, iscsiUsername, iscsiPassword string) {
	// check target abnormal return
	if vol.Spec.IscsiTarget == "" {
		return
	}

	err := mount.UmountVolume(&vol, info.VolumeInfo.MountPath, iscsiUsername, iscsiPassword, nil, false)
	if err != nil {
		logger.StdLog.Errorf("recovery disk %s unmount volume with error %v", vol.Name, err)
	}

	err = mount.MountVolume(&vol, info, iscsiUsername, iscsiPassword)
	if err != nil {
		logger.StdLog.Errorf("recovery disk %s iscsi session node %s pod %s with error %v",
			vol.Name, info.PodInfo.NodeId, info.PodInfo.Name, err)
	}

}

// CheckAndRecoveryDiskIscsi check disk status and do iscsi recovery
func CheckAndRecoveryDiskIscsi(vol apis.Volume, iscsiUsername, iscsiPassword string, targetMap map[string]bool) {
	target := vol.Spec.IscsiTarget

	// check target already exist or abnormal return
	if vol.Spec.IscsiTarget == "" || targetMap[target] {
		return
	}

	_, err := iscsi.CreateTarget(target)
	if err != nil {
		logger.StdLog.Errorf("CheckAndRecoveryDisk: CreateTarget %s error %v", target, err)
	}

	// check ACL
	err = CreateTargetAcl(vol.Namespace, vol.Spec.IscsiTarget, iscsiUsername, iscsiPassword)
	if err != nil {
		logger.StdLog.Error(err, fmt.Sprintf("CheckAndRecoveryDisk: CreateTargetAcl %v", err))
	}

	// public block device
	device := getVolumeDevice(&vol)
	_, err = iscsi.PublicBlockDevice(vol.Name, device)
	if err != nil {
		logger.StdLog.Error(err, fmt.Sprintf("PublicBlockDevice target %s, vol %s, device %s error: %v",
			vol.Spec.IscsiTarget, vol.Name, device, err))
	}

	// mount lun device
	_, err = iscsi.MountLun(vol.Spec.IscsiTarget, vol.Name)
	if err != nil {
		logger.StdLog.Error(err, fmt.Sprintf("MountLun target %s, vol %s,  error: %v",
			vol.Spec.IscsiTarget, vol.Name, err))
	}

}
