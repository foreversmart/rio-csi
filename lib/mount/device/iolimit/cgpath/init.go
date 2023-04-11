package cgpath

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"qiniu.io/rio-csi/logger"
	"strings"
)

var MountPoint string

func init() {
	mp, err := FindMountPoint()
	if err != nil {
		logger.StdLog.Error(err)
	}

	MountPoint = mp
}

// FindMountPoint returns the mount point where the cgroup
// mountpoints are mounted in a single hiearchy
func FindMountPoint() (string, error) {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var (
			text      = scanner.Text()
			fields    = strings.Split(text, " ")
			numFields = len(fields)
		)
		if numFields < 10 {
			continue
		}
		if fields[numFields-3] == "cgroup" {
			return filepath.Dir(fields[4]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("ErrMountPointNotExist")
}
