package iscsi

import (
	"errors"
	"os"
	"strings"
)

const initiatorNameFile = "/etc/iscsi/initiatorname.iscsi"

func GetInitiatorName() (name string, err error) {
	content, err := os.ReadFile(initiatorNameFile)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "#") {
			continue
		}

		if strings.HasPrefix(l, "InitiatorName=") {
			return strings.TrimPrefix(l, "InitiatorName="), nil
		}
	}

	return "", errors.New("InitiatorName not found")
}
