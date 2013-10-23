package main

import (
	"net"
)

func (this *Server) Ps_List(conn net.Conn, applicationName string) error {
	return this.WithApplication(applicationName, func(app *Application, cfg *Config) error {
		str := ""
		for process, numDynos := range app.Processes {
			dynos, err := this.GetRunningDynos(app.Name, process)
			if err != nil {
				Logf(conn, "Error: %v (process was '%v')", err, process)
				continue
			}
			Logf(conn, "=== %v: dyno scale=%v, actual=%v\n", process, numDynos, len(dynos))
			for _, dyno := range dynos {
				Logf(conn, "%v @ %v [%v:%v]\n", process, dyno.Version, dyno.Host, dyno.Port)
			}
			Logf(conn, "\n")
		}
		return Send(conn, Message{Log, str})
	})
}

// e.g. ps:scale web=12 worker=12 scheduler=1
func (this *Server) Ps_Scale(conn net.Conn, applicationName string, args map[string]string) error {
	return this.Rescale(conn, applicationName, args)
}
