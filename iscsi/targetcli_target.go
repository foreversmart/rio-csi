package iscsi

import (
	"fmt"
	"strings"
	"time"
)

const targetFormat = "iqn.%s.rio-csi:%s.%s"
const targetTimeFormat = "2006-01"

func SetUpTarget(target string) (string, error) {
	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(createCmd, target)
	_, err := cmd.Exec()
	if err != nil {
		return "", err
	}

	return target, nil
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
