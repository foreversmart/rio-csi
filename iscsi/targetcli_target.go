package iscsi

import (
	"fmt"
	"strings"
	"time"
)

func SetUpTarget(group, name string) (string, error) {
	now := time.Now()
	target := fmt.Sprintf("iqn.%d-%d.%s.srv:rio.%s", now.Year(), now.Month(), group, name)

	cmd := NewExecCmd()
	cmd.Add(openIscsiDir)
	cmd.AddFormat(createCmd, target)
	_, err := cmd.Exec()
	if err != nil {
		return "", err
	}

	return target, nil
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
