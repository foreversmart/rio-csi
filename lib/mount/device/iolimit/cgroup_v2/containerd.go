package cgroup_v2

import (
	"errors"
	"qiniu.io/rio-csi/lib/lvm/common/helpers"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/params"
	"strings"
)

type ContainerdPath struct {
	PodUid string
}

func (p *ContainerdPath) CGroupPath() (string, error) {
	kubepodsCGPath := params.BaseCgroupPath + "/kubepods.slice"
	podSuffix := p.PodSuffix()
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
	return "", errors.New("CGroup Path not found for pod with Uid: " + p.PodUid)
}

func (p *ContainerdPath) PodSuffix() string {
	return "pod" + strings.ReplaceAll(p.PodUid, "-", "_")
}
