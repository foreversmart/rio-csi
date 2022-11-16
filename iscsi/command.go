package iscsi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

type ExecCmd struct {
	cmds []string
}

func NewExecCmd() *ExecCmd {
	return &ExecCmd{
		cmds: make([]string, 0, 5),
	}
}

func (c *ExecCmd) Add(cmd string) {
	c.cmds = append(c.cmds, cmd+"\n")
}

func (c *ExecCmd) AddFormat(cmd string, params ...interface{}) {
	c.cmds = append(c.cmds, fmt.Sprintf(cmd+"\n", params...))
}

func (c *ExecCmd) String() string {
	b := &bytes.Buffer{}
	for _, cmd := range c.cmds {
		b.WriteString(cmd)
	}
	return b.String()
}

func (c *ExecCmd) Exec() (res string, err error) {
	cmd := exec.Command("targetcli")

	in, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	defer in.Close()

	go func() {
		io.WriteString(in, c.String())
		// auto exit
		io.WriteString(in, exitCmd)
	}()

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		err = errors.New(fmt.Sprint(err) + ": " + stderr.String())
		return "", err
	}

	if len(stderr.String()) > 0 {
		err = errors.New(stderr.String())
		return "", err
	}

	res = out.String()
	return
}
