package cgroup_v2

import (
	"os"
	"qiniu.io/rio-csi/lib/lvm/common/errors"
	"qiniu.io/rio-csi/lib/lvm/common/helpers"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/cgpath"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/params"
	"strconv"
)

type Limit struct {
	DeviceName       string
	PodUid           string
	ContainerRuntime string
	IOLimit          *params.IOThrottle
}

func NewLimit(device, podUid, containerRuntime string, ioLimit *params.IOThrottle) *Limit {
	return &Limit{
		DeviceName:       device,
		PodUid:           podUid,
		ContainerRuntime: containerRuntime,
		IOLimit:          ioLimit,
	}
}

func (l *Limit) SetIOLimits() error {
	cgroupPath, err := l.getIoMaxCGroupPath()
	if err != nil {
		return err
	}

	deviceNumber, err := params.GetDeviceNumber(l.DeviceName)
	if err != nil {
		return errors.New("Device Major:Minor numbers could not be obtained")
	}

	line := l.GetIOLimitsStr(deviceNumber)

	err = os.WriteFile(cgroupPath, []byte(line), 0600)
	return err
}

func (l *Limit) getIoMaxCGroupPath() (string, error) {
	if !helpers.IsValidUUID(l.PodUid) {
		return "", errors.New("Expected PodUid in UUID format, Got " + l.PodUid)
	}

	absPath, _, err := cgpath.PodCGroupPath(l.PodUid, l.ContainerRuntime)
	if err != nil {
		return "", err
	}

	ioMaxFile := absPath + "/io.max"
	if !helpers.FileExists(ioMaxFile) {
		return "", errors.New("io.max file is not present in pod CGroup")
	}

	return ioMaxFile, nil
}

func (l *Limit) GetIOLimitsStr(deviceNumber *params.DeviceNumber) string {
	line := strconv.FormatUint(deviceNumber.Major, 10) + ":" + strconv.FormatUint(deviceNumber.Minor, 10)
	if l.IOLimit.ReadIOPS != 0 {
		line += " riops=" + strconv.FormatUint(l.IOLimit.ReadIOPS, 10)
	}
	if l.IOLimit.WriteIOPS != 0 {
		line += " wiops=" + strconv.FormatUint(l.IOLimit.WriteIOPS, 10)
	}
	if l.IOLimit.ReadBps != 0 {
		line += " rbps=" + strconv.FormatUint(l.IOLimit.ReadBps, 10)
	}
	if l.IOLimit.WriteBps != 0 {
		line += " wbps=" + strconv.FormatUint(l.IOLimit.WriteBps, 10)
	}
	return line
}
