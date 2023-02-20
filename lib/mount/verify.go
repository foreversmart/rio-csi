package mount

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/logger"
)

func verifyMountRequest(vol *apis.Volume, mountpath string) (bool, error) {
	if len(mountpath) == 0 {
		return false, status.Error(codes.InvalidArgument, "verifyMount: mount path missing in request")
	}

	if vol.Finalizers == nil {
		return false, status.Error(codes.Internal, "verifyMount: volume is not ready to be mounted")
	}

	devicePath, err := GetVolumeDevPath(vol)
	if err != nil {
		logger.StdLog.Errorf("can not get device for volume:%s dev %s err: %v",
			vol.Name, devicePath, err.Error())
		return false, status.Errorf(codes.Internal, "verifyMount: GetVolumePath failed %s", err.Error())
	}

	/*
	 * This check is the famous *Wall Of North*
	 * It will not let the volume to be mounted
	 * at more than two places. The volume should
	 * be unmounted before proceeding to the mount
	 * operation.
	 */
	currentMounts, err := GetMounts(devicePath)
	if err != nil {
		logger.StdLog.Errorf("can not get mounts for volume:%s dev %s err: %v",
			vol.Name, devicePath, err.Error())
		return false, status.Errorf(codes.Internal, "verifyMount: Getmounts failed %s", err.Error())
	} else if len(currentMounts) >= 1 {
		// if device is already mounted at the mount point, return successful
		for _, mp := range currentMounts {
			if mp == mountpath {
				return true, nil
			}
		}

		// if it is not a shared volume, then it should not mounted to more than one path
		if vol.Spec.Shared != "yes" {
			logger.StdLog.Errorf(
				"can not mount, volume:%s already mounted dev %s mounts: %v",
				vol.Name, devicePath, currentMounts,
			)
			return false, status.Errorf(codes.Internal, "verifyMount: device already mounted at %s", currentMounts)
		}
	}
	return false, nil
}
