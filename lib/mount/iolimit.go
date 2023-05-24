package mount

import (
	"errors"
	"math"
	apis "qiniu.io/rio-csi/api/rio/v1"
	"qiniu.io/rio-csi/lib/mount/device/iolimit"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/params"
	"qiniu.io/rio-csi/logger"
	"strconv"
	"strings"
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

	ioThrottle := calcIOThrottle(vol.Spec.VolGroup, capacityGB)

	logger.StdLog.Infof("Setting iolimits for podUId %s, device %s: riops=%v, wiops=%v, rbps=%v, wbps=%v",
		podLVInfo.UID, devicePath, ioThrottle.ReadIOPS, ioThrottle.WriteIOPS, ioThrottle.ReadBps, ioThrottle.WriteBps,
	)
	err = iolimit.SetIOLimits(&iolimit.Request{
		DeviceName:       devicePath,
		PodUid:           podLVInfo.UID,
		ContainerRuntime: getContainerRuntime(),
		IOThrottle:       ioThrottle,
	})
	if err != nil {
		logger.StdLog.Errorf("setting iolimits pod %s  device %s with error %v ", podLVInfo.UID, devicePath, err)
		return err
	}
	return nil
}

// calcIOThrottle formula is min{(BaseBpsLimit + ReadBpsLimit * GB), MaxBpsLimit}
func calcIOThrottle(vgName string, capacityGB uint64) *params.IOThrottle {
	riops := getRatePerGB(vgName, riopsPerGBMap) * capacityGB
	wiops := getRatePerGB(vgName, wiopsPerGBMap) * capacityGB
	baseIops := getRatePerGB(vgName, baseBpsPerGBMap) * capacityGB
	maxIops := getRatePerGB(vgName, maxBpsPerGBMap) * capacityGB

	rbps := getRatePerGB(vgName, rbpsPerGBMap) * capacityGB
	wbps := getRatePerGB(vgName, wbpsPerGBMap) * capacityGB
	baseBps := getRatePerGB(vgName, baseBpsPerGBMap) * capacityGB
	maxBps := getRatePerGB(vgName, maxBpsPerGBMap) * capacityGB

	return &params.IOThrottle{
		ReadIOPS:  min(baseIops+riops, maxIops),
		WriteIOPS: min(baseIops+wiops, maxIops),
		ReadBps:   min(baseBps+rbps, maxBps),
		WriteBps:  min(baseBps+wbps, maxBps),
	}
}

func min(a, b uint64) uint64 {
	if a > b {
		return b
	}

	return a
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

func getContainerRuntime() string {
	rwlock.RLock()
	defer rwlock.RUnlock()
	return containerRuntime
}
