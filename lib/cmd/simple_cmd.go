package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

type SimpleCmd struct {
	cmd    string
	params []string
}

func NewSimpleCmd(cmd string, params ...string) *SimpleCmd {
	c := &SimpleCmd{
		cmd:    cmd,
		params: params,
	}

	return c
}

func (c *SimpleCmd) AddParam(param string) {
	c.params = append(c.params, param)
}

func (c *SimpleCmd) AddFormatParam(format string, params ...interface{}) {
	c.params = append(c.params, fmt.Sprintf(format, params...))
}

func (c *SimpleCmd) Exec() (res string, err error) {
	cmd := exec.Command(c.cmd, c.params...)

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
