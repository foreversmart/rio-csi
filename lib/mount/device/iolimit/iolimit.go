package iolimit

import (
	"qiniu.io/rio-csi/lib/lvm/common/errors"
	"qiniu.io/rio-csi/lib/lvm/common/helpers"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/cgpath"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/cgroup_v1"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/cgroup_v2"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/params"
)

type IOLimiter interface {
	SetIOLimits() error
}

type Request struct {
	DeviceName       string
	PodUid           string
	ContainerRuntime string
	IOThrottle       *params.IOThrottle
}

// SetIOLimits sets iops, bps limits for a pod with uid podUid for accessing a device named deviceName
// provided that the underlying cgroup used for pod namespacing is cgroup2 (cgroup v2)
func SetIOLimits(request *Request) error {
	var (
		limit IOLimiter
	)
	if cgpath.MountPoint == "" {
		return errors.New("mount point does not found in system")
	}

	err := checkCgroupV2()
	if err == nil {
		limit = cgroup_v2.NewLimit(request.DeviceName, request.PodUid, request.ContainerRuntime, request.IOThrottle)
	}

	err = checkCgroupV1()
	if err == nil {
		limit = cgroup_v1.NewLimit(request.DeviceName, request.PodUid, request.ContainerRuntime, request.IOThrottle)
	}

	return limit.SetIOLimits()
}

func checkCgroupV1() error {
	return nil
}

func checkCgroupV2() error {
	if !helpers.FileExists(cgpath.MountPoint + "/cgroup.controllers") {
		return errors.New("CGroupV2 not enabled")
	}
	return nil
}
