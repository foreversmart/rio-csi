package cgroup_v2

import (
	"errors"
)

type CGPather interface {
	CGroupPath() (string, error)
}

func getPodCGroupPath(podUid string, cruntime string) (string, error) {
	var pather CGPather
	switch cruntime {
	case "containerd":
		pather = &ContainerdPath{
			PodUid: podUid,
		}
	default:
		return "", errors.New(cruntime + " runtime support is not present")
	}

	path, err := pather.CGroupPath()
	if err != nil {
		return "", err
	}
	return path, nil
}
