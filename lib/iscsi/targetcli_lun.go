package iscsi

import (
	"errors"
	"strings"
)

type LunDevice struct {
	Id     string // lun id eg. lun0
	Disk   string // path
	Device string // device path
}

// MountLun mount device as lun Only support block device
func MountLun(target, disk string) (string, error) {
	disk = "/backstores/block/" + disk
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(cdCmd, target)
	cmd.Add(openLunsDir)
	cmd.AddFormat(createCmd, disk)

	Lock.Lock()
	defer Lock.Unlock()

	out, err := cmd.Exec()
	if err != nil {
		return "", err
	}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Created LUN ") && strings.HasSuffix(line, ".") {
			items := strings.Split(line, "Created LUN ")
			s := strings.TrimSpace(items[1])
			s = strings.TrimSuffix(s, ".")
			return s, nil
		}
	}

	return "", errors.New("cant get lun id")
}

// UnmountLun mount device as lun Only support block device
func UnmountLun(target, lunId string) (string, error) {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(cdCmd, target)
	cmd.Add(openLunsDir)
	cmd.AddFormat(deleteCmd, lunId)

	Lock.Lock()
	defer Lock.Unlock()

	res, err := cmd.Exec()
	if err != nil {
		if strings.Contains(err.Error(), "Invalid LUN") || strings.Contains(err.Error(), "No such path") {
			return res, nil
		}
	}

	return res, err
}

func LunList(target string) ([]*LunDevice, error) {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(cdCmd, target)
	cmd.Add(openLunsDir)
	cmd.Add(lsCmd)

	Lock.Lock()
	defer Lock.Unlock()

	out, err := cmd.Exec()
	if err != nil {
		return nil, err
	}

	res := make([]*LunDevice, 0, 5)
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		pointLoc := strings.Index(line, " ......")
		leftBracketLoc := strings.Index(line, "[")
		rightBracketLoc := strings.Index(line, "]")
		if strings.HasPrefix(line, "  o- ") && pointLoc > 0 && leftBracketLoc > 0 && rightBracketLoc > 0 {
			t := line[:pointLoc]
			t = strings.TrimPrefix(line, "  o- ")
			lun := &LunDevice{
				Id: t,
			}

			items := strings.Split(line[leftBracketLoc+1:rightBracketLoc], " ")
			if len(items) == 2 {
				lun.Disk = strings.TrimPrefix(items[0], "block/")
				lun.Device = items[1]
			}

			res = append(res, lun)
		}
	}

	return res, nil

}
