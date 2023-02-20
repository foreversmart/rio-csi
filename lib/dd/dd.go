package dd

import "qiniu.io/rio-csi/lib/cmd"

var (
	mainCmd = "dd"
	dumpCmd = "if=%s of=%s"
)

func DiskDump(in, out, args string) error {
	c := cmd.NewExecCmd(mainCmd)
	c.AddFormat(dumpCmd, in, out)
	_, err := c.Exec()
	return err
}
