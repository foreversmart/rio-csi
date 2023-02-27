package dd

import (
	"qiniu.io/rio-csi/lib/cmd"
)

var (
	mainCmd     = "dd"
	inputParam  = "if=%s"
	outputParam = "of=%s"
	noneStatus  = "status=none"
)

// DiskDump exec dd command to dump disk Partition
// 'status=LEVEL'
//
//	Transfer information is normally output to stderr upon receipt of
//	the 'INFO' signal or when 'dd' exits.  Specifying LEVEL will adjust
//	the amount of information printed, with the last LEVEL specified
//	taking precedence.
//
//	'none'
//	     Do not print any informational or warning messages to stderr.
//	     Error messages are output as normal.
//
//	'noxfer'
//	     Do not print the final transfer rate and volume statistics
//	     that normally make up the last status line.
//
//	'progress'
//	     Print the transfer rate and volume statistics on stderr, when
//	     processing each input block.  Statistics are output on a
//	     single line at most once every second, but updates can be
//	     delayed when waiting on I/O.
//
// so dd will print some informational or warning messages to stderr when status is not 'none'
// Note some dd release version may not support status = 'none'
func DiskDump(in, out string) error {
	c := cmd.NewSimpleCmd(mainCmd)
	c.AddFormatParam(inputParam, in)
	c.AddFormatParam(outputParam, out)
	c.AddParam(noneStatus)
	_, err := c.Exec()
	return err
}
