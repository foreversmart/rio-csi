package iscsi

// PublicBlockDevice publish device as block device
func PublicBlockDevice(disk, device string) (string, error) {
	cmd := NewExecCmd()
	cmd.Add(openBlockDir)
	cmd.AddFormat(createBlockCmd, disk, device)
	return cmd.Exec()
}

// UnPublicBlockDevice publish device as block device
func UnPublicBlockDevice(disk string) (string, error) {
	cmd := NewExecCmd()
	cmd.Add(openBlockDir)
	cmd.AddFormat(deleteCmd, disk)
	return cmd.Exec()
}
