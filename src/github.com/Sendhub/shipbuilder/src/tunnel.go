package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type (
	Tunnel struct {
		*exec.Cmd
	}
)

func OpenTunnel() (Tunnel, error) {
	fmt.Printf("Client connecting via '%v'..\n", sshHost)

	sshArgs := append(defaultSshParametersList, "-N", "-L", "9999:127.0.0.1:9999")
	if len(sshKey) > 0 {
		sshArgs = append(sshArgs, "-i", sshKey)
	}
	sshArgs = append(sshArgs, "-v", sshHost)

	cmd := exec.Command("ssh", sshArgs...)
	t := Tunnel{cmd}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return t, err
	}
	defer stderr.Close()
	err = cmd.Start()
	if err != nil {
		return t, err
	}

	wait := make(chan error)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			//fmt.Printf("SSH DEBUG: %v\n", scanner.Text())
			if strings.Contains(scanner.Text(), "Entering interactive session.") {
				wait <- nil
				break
			}
		}
		err := scanner.Err()
		if err != nil {
			wait <- err
		}
	}()
	return t, <-wait
}

func (this Tunnel) Close() error {
	err := this.Process.Signal(os.Interrupt)
	if err != nil {
		err = this.Process.Signal(os.Kill)
	}
	return err
}
