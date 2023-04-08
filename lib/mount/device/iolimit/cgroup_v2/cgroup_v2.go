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
	"io/ioutil"
	"os"
	"qiniu.io/rio-csi/lib/lvm/common/errors"
	"qiniu.io/rio-csi/lib/lvm/common/helpers"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/params"
	"strconv"
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

	deviceNumber, err := getDeviceNumber(l.DeviceName)
	if err != nil {
		return errors.New("Device Major:Minor numbers could not be obtained")
	}

	line := getIOLimitsStr(deviceNumber, l.IOLimit)

	err = os.WriteFile(cgroupPath, []byte(line), 0600)
	return err
}

func (l *Limit) getIoMaxCGroupPath() (string, error) {
	if !helpers.IsValidUUID(l.PodUid) {
		return "", errors.New("Expected PodUid in UUID format, Got " + l.PodUid)
	}

	podCGPath, err := getPodCGroupPath(l.PodUid, l.ContainerRuntime)
	if err != nil {
		return "", err
	}

	ioMaxFile := podCGPath + "/io.max"
	if !helpers.FileExists(ioMaxFile) {
		return "", errors.New("io.max file is not present in pod CGroup")
	}

	return ioMaxFile, nil
}

func getIOLimitsStr(deviceNumber *params.DeviceNumber, ioMax *params.IOMax) string {
	line := strconv.FormatUint(deviceNumber.Major, 10) + ":" + strconv.FormatUint(deviceNumber.Minor, 10)
	if ioMax.Riops != 0 {
		line += " riops=" + strconv.FormatUint(ioMax.Riops, 10)
	}
	if ioMax.Wiops != 0 {
		line += " wiops=" + strconv.FormatUint(ioMax.Wiops, 10)
	}
	if ioMax.Rbps != 0 {
		line += " rbps=" + strconv.FormatUint(ioMax.Rbps, 10)
	}
	if ioMax.Wbps != 0 {
		line += " wbps=" + strconv.FormatUint(ioMax.Wbps, 10)
	}
	return line
}

func setIOLimits(request *params.ValidRequest) error {
	line := getIOLimitsStr(request.DeviceNumber, request.IOMax)
	err := ioutil.WriteFile(request.FilePath, []byte(line), 0600)
	return err
}
