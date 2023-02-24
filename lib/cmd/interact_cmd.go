package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

type InteractCmd struct {
	mainCmd string
	cmds    []string
	exitCmd string
}

func NewInteractCmd(mainCmd string, exitCmd ...string) *InteractCmd {
	c := &InteractCmd{
		mainCmd: mainCmd,
		cmds:    make([]string, 0, 5),
	}

	if len(exitCmd) > 0 {
		c.exitCmd = exitCmd[0]
	}

	return c
}

func (c *InteractCmd) Add(cmd string) {
	c.cmds = append(c.cmds, cmd+"\n")
}

func (c *InteractCmd) AddFormat(cmd string, params ...interface{}) {
	c.cmds = append(c.cmds, fmt.Sprintf(cmd+"\n", params...))
}

func (c *InteractCmd) String() string {
	b := &bytes.Buffer{}
	for _, cmd := range c.cmds {
		b.WriteString(cmd)
	}
	return b.String()
}

func (c *InteractCmd) Exec() (res string, err error) {
	cmd := exec.Command(c.mainCmd)

	in, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	defer in.Close()

	go func() {
		io.WriteString(in, c.String())
		// auto exit
		if len(c.exitCmd) > 0 {
			io.WriteString(in, c.exitCmd)
		}
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
