package cgpath

import (
	"errors"
	"qiniu.io/rio-csi/lib/lvm/common/helpers"
	"qiniu.io/rio-csi/lib/mount/device/iolimit/params"
	"strings"
)

type ContainerdPath struct {
	PodUid string
}

// PodCGroupPath return pod cgroup abs path and relative path
func (p *ContainerdPath) PodCGroupPath() (string, string, error) {
	rootPath := params.BaseCgroupPath
	kubepodsPath := "/kubepods.slice"
	podSuffix := p.PodSuffix()
	relativePath := kubepodsPath + "/kubepods-" + podSuffix + ".slice"
	absPath := rootPath + relativePath
	if helpers.DirExists(absPath) {
		return absPath, relativePath, nil
	}
	relativePath = kubepodsPath + "/kubepods-besteffort.slice/kubepods-besteffort-" + podSuffix + ".slice"
	absPath = rootPath + relativePath
	if helpers.DirExists(rootPath + relativePath) {
		return absPath, relativePath, nil
	}
	relativePath = kubepodsPath + "/kubepods-burstable.slice/kubepods-burstable-" + podSuffix + ".slice"
	absPath = rootPath + relativePath
	if helpers.DirExists(rootPath + relativePath) {
		return absPath, relativePath, nil
	}
	return "", "", errors.New("CGroup Path not found for pod with Uid: " + p.PodUid)
}

func (p *ContainerdPath) PodSuffix() string {
	return "pod" + strings.ReplaceAll(p.PodUid, "-", "_")
}
