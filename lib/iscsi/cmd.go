package iscsi

import "qiniu.io/rio-csi/lib/cmd"

const (
	targetCliCmd = "targetcli"
)

func NewExecCmd() *cmd.ExecCmd {
	return cmd.NewExecCmd(targetCliCmd, exitCmd)
}
