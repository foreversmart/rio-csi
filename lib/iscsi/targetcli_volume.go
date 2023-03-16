package iscsi

import (
	"fmt"
	"strings"
)

// PublicBlockDevice publish device as block device
func PublicBlockDevice(disk, device string) (string, error) {
	cmd := NewExecCmd()
	cmd.Add(openBlockDir)
	cmd.AddFormat(createBlockCmd, disk, device)

	Lock.Lock()
	defer Lock.Unlock()

	res, err := cmd.Exec()
	if err != nil {
		alreadyExistErr := fmt.Sprintf("Storage object block/%s exists", disk)
		if err.Error() == alreadyExistErr {
			return res, nil
		}
	}

	return res, err
}

// UnPublicBlockDevice publish device as block device
func UnPublicBlockDevice(disk string) (string, error) {
	cmd := NewExecCmd()
	cmd.Add(openBlockDir)
	cmd.AddFormat(deleteCmd, disk)

	Lock.Lock()
	defer Lock.Unlock()

	res, err := cmd.Exec()
	if err != nil {
		if strings.Contains(err.Error(), "No storage object") {
			return res, nil
		}
	}

	return res, err
}
