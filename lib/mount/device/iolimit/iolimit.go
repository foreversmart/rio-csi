package iolimit

import (
	"qiniu.io/rio-csi/lib/lvm/common/errors"
	"qiniu.io/rio-csi/lib/lvm/common/helpers"
)

const (
	baseCgroupPath = "/sys/fs/cgroup"
)

// SetIOLimits sets iops, bps limits for a pod with uid podUid for accessing a device named deviceName
// provided that the underlying cgroup used for pod namespacing is cgroup2 (cgroup v2)
func SetIOLimits(request *Request) error {
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

func checkCgroupV2() error {
	if !helpers.FileExists(baseCgroupPath + "/cgroup.controllers") {
		return errors.New("CGroupV2 not enabled")
	}
	return nil
}
