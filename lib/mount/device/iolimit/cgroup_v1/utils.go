package cgroup_v1

import (
	"github.com/opencontainers/runtime-spec/specs-go"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/params"
)

func getThrottleLimit(devNumber *params.DeviceNumber, rate uint64) specs.LinuxThrottleDevice {
	t := specs.LinuxThrottleDevice{}
	t.Major = int64(devNumber.Major)
	t.Minor = int64(devNumber.Minor)
	t.Rate = rate
	return t
}
