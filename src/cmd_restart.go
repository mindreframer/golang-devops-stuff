package main

import (
	"fmt"
	"net"
)

func (this *Server) Restart_App(conn net.Conn, applicationName string) error {
	return this.WithApplication(applicationName, func(app *Application, cfg *Config) error {
		titleLogger, _ := this.getTitleAndDimLoggers(conn)
		fmt.Fprintf(titleLogger, "=== Restarting %v\n", app.Name)
		return this.Redeploy(conn, applicationName)
	})
}
