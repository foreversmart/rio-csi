/*
Copyright 2020 The OpenEBS Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cgroup_v2

import (
	"os"
	"qiniu.io/rio-csi/lib/lvm/common/errors"
	"qiniu.io/rio-csi/lib/lvm/common/helpers"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/cgpath"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/params"
	"strconv"
	"syscall"
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
	cgroupPath, err := l.getIoMaxCGroupPath()
	if err != nil {
		return err
	}

	deviceNumber, err := l.GetDeviceNumber()
	if err != nil {
		return errors.New("Device Major:Minor numbers could not be obtained")
	}

	line := l.GetIOLimitsStr(deviceNumber)

	err = os.WriteFile(cgroupPath, []byte(line), 0600)
	return err
}

func (l *Limit) GetDeviceNumber() (*params.DeviceNumber, error) {
	stat := syscall.Stat_t{}
	if err := syscall.Stat(l.DeviceName, &stat); err != nil {
		return nil, err
	}
	return &params.DeviceNumber{
		Major: uint64(stat.Rdev / 256),
		Minor: uint64(stat.Rdev % 256),
	}, nil
}

func (l *Limit) getIoMaxCGroupPath() (string, error) {
	if !helpers.IsValidUUID(l.PodUid) {
		return "", errors.New("Expected PodUid in UUID format, Got " + l.PodUid)
	}

	podCGPath, err := cgpath.PodCGroupPath(l.PodUid, l.ContainerRuntime)
	if err != nil {
		return "", err
	}

	ioMaxFile := podCGPath + "/io.max"
	if !helpers.FileExists(ioMaxFile) {
		return "", errors.New("io.max file is not present in pod CGroup")
	}

	return ioMaxFile, nil
}

func (l *Limit) GetIOLimitsStr(deviceNumber *params.DeviceNumber) string {
	line := strconv.FormatUint(deviceNumber.Major, 10) + ":" + strconv.FormatUint(deviceNumber.Minor, 10)
	if l.IOLimit.Riops != 0 {
		line += " riops=" + strconv.FormatUint(l.IOLimit.Riops, 10)
	}
	if l.IOLimit.Wiops != 0 {
		line += " wiops=" + strconv.FormatUint(l.IOLimit.Wiops, 10)
	}
	if l.IOLimit.Rbps != 0 {
		line += " rbps=" + strconv.FormatUint(l.IOLimit.Rbps, 10)
	}
	if l.IOLimit.Wbps != 0 {
		line += " wbps=" + strconv.FormatUint(l.IOLimit.Wbps, 10)
	}
	return line
}
