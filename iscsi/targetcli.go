package iscsi

import (
	"fmt"
	"strings"
	"time"
)

var (
	openIscsiDir = "cd /iscsi"
	openBlockDir = "cd /backstores/block"

	// must under target dir
	openAclsDir = "cd tpg1/acls"
	openLunsDir = "cd tpg1/luns"

	lsCmd = "ls"

	createCmd      = "create %s"
	createBlockCmd = "create %s %s"
	cdCmd          = "cd %s"
	setUserIDCmd   = "set auth userid=%s"
	setPasswordCmd = "set auth password=%s"

	exitCmd = "exit\n"
)

func SetUpTarget(group, name string) (string, error) {
	now := time.Now()
	target := fmt.Sprintf("iqn.%d-%d.%s.srv:rio.%s", now.Year(), now.Month(), group, name)

	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(createCmd, target)
	return cmd.Exec()
}

func ListTarget() ([]string, error) {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.Add(lsCmd)
	out, err := cmd.Exec()
	if err != nil {
		return nil, err
	}

	targetList := make([]string, 0, 5)
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		pointLoc := strings.Index(line, " ......")
		if strings.HasPrefix(line, "  o- ") && pointLoc > 0 {
			t := line[:pointLoc]
			t = strings.TrimPrefix(t, "  o- ")
			targetList = append(targetList, t)
		}
	}

	return targetList, nil
}

func SetUpTargetAcl(target, username, password string) (string, error) {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(cdCmd, target)
	cmd.Add(openAclsDir)
	// create acls
	cmd.AddFormat(createCmd, target)
	cmd.AddFormat(cdCmd, target)
	// set username and password
	cmd.AddFormat(setUserIDCmd, username)
	cmd.AddFormat(setPasswordCmd, password)
	return cmd.Exec()
}

// PublicBlockDevice publish device as block device
func PublicBlockDevice(disk, device string) (string, error) {
	cmd := NewExecCmd()
	cmd.Add(openBlockDir)
	cmd.AddFormat(createBlockCmd, disk, device)
	return cmd.Exec()

}

// MountLun mount device as lun Only support block device
func MountLun(target, disk string) (string, error) {
	disk = "/backstores/block/" + disk
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(cdCmd, target)
	cmd.Add(openLunsDir)
	cmd.AddFormat(createCmd, disk)
	return cmd.Exec()
}
