package main

import (
	"net"
	"regexp"

	"github.com/Sendhub/logserver"
)

func (this *Server) Logs_Get(conn net.Conn, applicationName, process, filter string) error {
	msgLogger := NewMessageLogger(conn)
	var r *regexp.Regexp

	if filter != "" {
		var err error
		r, err = regexp.Compile(filter)
		if err != nil {
			return err
		}
	}

	return this.LogServer.StartListener(msgLogger, log.EntryFilter{
		Application: applicationName,
		Process:     process,
		Data:        r,
	})
}
