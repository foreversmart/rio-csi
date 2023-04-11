package cgpath

import (
	"errors"
)

type CGPather interface {
	// PodCGroupPath will return pod cgroup abs path and relative path
	PodCGroupPath() (string, string, error)
}

func PodCGroupPath(podUid string, cruntime string) (string, string, error) {
	var pather CGPather
	switch cruntime {
	case "containerd":
		pather = &ContainerdPath{
			PodUid: podUid,
		}
	default:
		return "", "", errors.New(cruntime + " runtime support is not present")
	}

	absPath, relativePath, err := pather.PodCGroupPath()
	if err != nil {
		return "", "", err
	}
	return absPath, relativePath, nil
}
