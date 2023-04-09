package cgroup_v2

import (
	"errors"
	"qiniu.io/rio-csi/lib/lvm/common/helpers"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/params"
	"strings"
)

func getPodCGroupPath(podUid string, cruntime string) (string, error) {
	switch cruntime {
	case "containerd":
		path, err := getContainerdCGPath(podUid)
		if err != nil {
			return "", err
		}
		return path, nil
	default:
		return "", errors.New(cruntime + " runtime support is not present")
	}

}

func getContainerdCGPath(podUid string) (string, error) {
	kubepodsCGPath := params.BaseCgroupPath + "/kubepods.slice"
	podSuffix := getContainerdPodCGSuffix(podUid)
	podCGPath := kubepodsCGPath + "/kubepods-" + podSuffix + ".slice"
	if helpers.DirExists(podCGPath) {
		return podCGPath, nil
	}
	podCGPath = kubepodsCGPath + "/kubepods-besteffort.slice/kubepods-besteffort-" + podSuffix + ".slice"
	if helpers.DirExists(podCGPath) {
		return podCGPath, nil
	}
	podCGPath = kubepodsCGPath + "/kubepods-burstable.slice/kubepods-burstable-" + podSuffix + ".slice"
	if helpers.DirExists(podCGPath) {
		return podCGPath, nil
	}
	return "", errors.New("CGroup Path not found for pod with Uid: " + podUid)
}

func getContainerdPodCGSuffix(podUid string) string {
	return "pod" + strings.ReplaceAll(podUid, "-", "_")
}
