package cgroup_v1

import (
	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/opencontainers/runtime-spec/specs-go"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/cgpath"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/params"
	"qiniu.io/rio-csi/logger"
)

type Limit struct {
	DeviceName       string
	PodUid           string
	ContainerRuntime string
	IOLimit          *params.IOMax
}

func NewLimit(device, podUid, containerRuntime string, ioLimit *params.IOMax) *Limit {
	return &Limit{
		DeviceName:       device,
		PodUid:           podUid,
		ContainerRuntime: containerRuntime,
		IOLimit:          ioLimit,
	}
}

func (l *Limit) SetIOLimits() error {
	_, relativePath, err := cgpath.PodCGroupPath(l.PodUid, l.ContainerRuntime)
	if err != nil {
		logger.StdLog.Error(err)
		return err
	}

	control, err := cgroup1.Load(cgroup1.StaticPath(relativePath))
	if err != nil {
		logger.StdLog.Error(err)
		return err
	}

	devNumber, err := params.GetDeviceNumber(l.DeviceName)
	if err != nil {
		logger.StdLog.Errorf("GetDeviceNumber %s with error %v", err)
		return err
	}

	return control.Update(&specs.LinuxResources{
		BlockIO: &specs.LinuxBlockIO{
			ThrottleReadBpsDevice:   []specs.LinuxThrottleDevice{getThrottleLimit(devNumber, l.IOLimit.Rbps)},
			ThrottleWriteBpsDevice:  []specs.LinuxThrottleDevice{getThrottleLimit(devNumber, l.IOLimit.Wbps)},
			ThrottleReadIOPSDevice:  []specs.LinuxThrottleDevice{getThrottleLimit(devNumber, l.IOLimit.Riops)},
			ThrottleWriteIOPSDevice: []specs.LinuxThrottleDevice{getThrottleLimit(devNumber, l.IOLimit.Wiops)},
		},
	})

}
