package iscsi

import "qiniu.io/rio-csi/lib/cmd"

const (
	targetCliCmd = "targetcli"
)

func NewExecCmd() *cmd.InteractCmd {
	return cmd.NewInteractCmd(targetCliCmd, exitCmd)
}
