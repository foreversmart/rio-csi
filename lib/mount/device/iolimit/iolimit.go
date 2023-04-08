package iolimit

import (
	"qiniu.io/rio-csi/lib/lvm/common/errors"
	"qiniu.io/rio-csi/lib/lvm/common/helpers"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/params"
)

const (
	baseCgroupPath = "/sys/fs/cgroup"
)

type IOLimiter interface {
	SetIOLimits(req *params.Request) error
}

// SetIOLimits sets iops, bps limits for a pod with uid podUid for accessing a device named deviceName
// provided that the underlying cgroup used for pod namespacing is cgroup2 (cgroup v2)
func SetIOLimits(request *params.Request) error {
	if !helpers.DirExists(baseCgroupPath) {
		return errors.New(baseCgroupPath + " does not exist")
	}
	if err := checkCgroupV2(); err != nil {
		return err
	}
	validRequest, err := validate(request)
	if err != nil {
		return err
	}
	err = setIOLimits(validRequest)
	return err
}

func checkCgroupV1() error {
	return nil
}

func checkCgroupV2() error {
	if !helpers.FileExists(baseCgroupPath + "/cgroup.controllers") {
		return errors.New("CGroupV2 not enabled")
	}
	return nil
}
