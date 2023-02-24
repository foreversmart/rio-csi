package dd

import "qiniu.io/rio-csi/lib/cmd"

var (
	mainCmd = "dd"
	dumpCmd = "if=%s of=%s"
)

func DiskDump(in, out string) error {
	c := cmd.NewInteractCmd(mainCmd)
	c.AddFormat(dumpCmd, in, out)
	_, err := c.Exec()
	return err
}
