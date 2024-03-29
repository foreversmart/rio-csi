package iscsi

import (
	"fmt"
	"strings"
	"time"
)

const targetFormat = "iqn.%s.rio-csi:%s.%s"
const targetTimeFormat = "2006-01"

func CreateTarget(target string) (string, error) {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(createCmd, target)

	Lock.Lock()
	defer Lock.Unlock()

	_, err := cmd.Exec()
	if err != nil {
		if strings.Contains(err.Error(), "Target already exists") {
			return target, nil
		}
		return "", err
	}

	return target, nil
}

func DeleteTarget(target string) error {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(deleteCmd, target)

	Lock.Lock()
	defer Lock.Unlock()

	_, err := cmd.Exec()
	if err != nil {
		// repeat delete
		if strings.Contains(err.Error(), "No such Target") || strings.Contains(err.Error(), "No such path") {
			return nil
		}

		return err
	}

	return nil

}

func GenerateTargetName(group, name string) string {
	now := time.Now()
	timeDate := now.Format(targetTimeFormat)
	return fmt.Sprintf(targetFormat, timeDate, group, name)
}

func ListTarget() ([]string, error) {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.Add(lsCmd)

	Lock.Lock()
	defer Lock.Unlock()

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
