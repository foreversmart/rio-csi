/*
Copyright © 2019 The OpenEBS Authors

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

package mount

import (
	"errors"
	"math"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/driver/config"
	iolimit2 "qiniu.io/rio-csi/lib/mount/device/iolimit"
	"qiniu.io/rio-csi/logger"
	"strconv"
	"strings"
	"sync"

	"k8s.io/klog"
)

var (
	set              = false
	ioLimitsEnabled  = false
	containerRuntime string
	riopsPerGB       map[string]uint64
	wiopsPerGB       map[string]uint64
	rbpsPerGB        map[string]uint64
	wbpsPerGB        map[string]uint64
	rwlock           sync.RWMutex
)

func setIOLimits(vol *apis.Volume, podLVInfo *PodInfo, devicePath string) error {
	if podLVInfo == nil {
		return errors.New("PodInfo is missing. Skipping setting IOLimits")
	}
	capacityBytes, err := strconv.ParseUint(vol.Spec.Capacity, 10, 64)
	if err != nil {
		logger.StdLog.Warn("error parsing Volume.Spec.Capacity. Skipping setting IOLimits", err)
		return err
	}
	capacityGB := uint64(math.Ceil(float64(capacityBytes) / (1024 * 1024 * 1024)))
	logger.StdLog.Infof("Capacity of device in GB: %v", capacityGB)
	riops := getRIopsPerGB(vol.Spec.VolGroup) * capacityGB
	wiops := getWIopsPerGB(vol.Spec.VolGroup) * capacityGB
	rbps := getRBpsPerGB(vol.Spec.VolGroup) * capacityGB
	wbps := getWBpsPerGB(vol.Spec.VolGroup) * capacityGB
	logger.StdLog.Infof("Setting iolimits for podUId %s, device %s: riops=%v, wiops=%v, rbps=%v, wbps=%v",
		podLVInfo.UID, devicePath, riops, wiops, rbps, wbps,
	)
	err = iolimit2.SetIOLimits(&iolimit2.Request{
		DeviceName:       devicePath,
		PodUid:           podLVInfo.UID,
		ContainerRuntime: getContainerRuntime(),
		IOLimit: &iolimit2.IOMax{
			Riops: riops,
			Wiops: wiops,
			Rbps:  rbps,
			Wbps:  wbps,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func isSet() bool {
	rwlock.RLock()
	defer rwlock.RUnlock()
	return set
}

func extractRateValues(rateVals *[]string) (map[string]uint64, error) {
	rate := map[string]uint64{}
	if rateVals == nil {
		return rate, nil
	}
	for _, kv := range *rateVals {
		parts := strings.Split(kv, ":")
		key := parts[0]
		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return nil, err
		}
		rate[key] = value
	}
	return rate, nil
}

func setValues(config *config.Config) {
	var err error
	riopsVals := config.RIopsLimitPerGB
	wiopsVals := config.WIopsLimitPerGB
	rbpsVals := config.RBpsLimitPerGB
	wbpsVals := config.WBpsLimitPerGB

	riopsPerGB, err = extractRateValues(riopsVals)
	if err != nil {
		klog.Warning("Read IOPS limit rates could not be extracted from config", err)
		riopsPerGB = map[string]uint64{}
	}

	wiopsPerGB, err = extractRateValues(wiopsVals)
	if err != nil {
		klog.Warning("Write IOPS limit rates could not be extracted from config", err)
		wiopsPerGB = map[string]uint64{}
	}

	rbpsPerGB, err = extractRateValues(rbpsVals)
	if err != nil {
		klog.Warning("Read BPS limit rates could not be extracted from config", err)
		rbpsPerGB = map[string]uint64{}
	}

	wbpsPerGB, err = extractRateValues(wbpsVals)
	if err != nil {
		klog.Warning("Write BPS limit rates could not be extracted from config", err)
		wbpsPerGB = map[string]uint64{}
	}
}

// SetIORateLimits sets io limit rates for the volume group (prefixes) provided in config
func SetIORateLimits(config *config.Config) {
	if isSet() {
		return
	}
	rwlock.Lock()
	defer rwlock.Unlock()

	ioLimitsEnabled = true
	containerRuntime = config.ContainerRuntime
	setValues(config)
	set = true
}

func getRatePerGB(vgName string, rateMap map[string]uint64) uint64 {
	rwlock.RLock()
	defer rwlock.RUnlock()
	if ptr, ok := rateMap[vgName]; ok {
		return ptr
	}
	for k, v := range rateMap {
		if strings.HasPrefix(vgName, k) {
			return v
		}
	}
	return uint64(0)
}

func getRIopsPerGB(vgName string) uint64 {
	return getRatePerGB(vgName, riopsPerGB)
}

func getWIopsPerGB(vgName string) uint64 {
	return getRatePerGB(vgName, wiopsPerGB)
}

func getRBpsPerGB(vgName string) uint64 {
	return getRatePerGB(vgName, rbpsPerGB)
}

func getWBpsPerGB(vgName string) uint64 {
	return getRatePerGB(vgName, wbpsPerGB)
}

func getContainerRuntime() string {
	rwlock.RLock()
	defer rwlock.RUnlock()
	return containerRuntime
}
