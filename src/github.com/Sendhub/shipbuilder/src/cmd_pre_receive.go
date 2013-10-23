package main

import (
	"net"
	"strings"
)

func (this *Server) PreReceive(conn net.Conn, dir, oldrev, newrev, ref string) error {
	// We only care about master
	if ref != "refs/heads/master" {
		return nil
	}
	return this.Deploy(conn, dir[strings.LastIndex(dir, "/")+1:], newrev)
}
