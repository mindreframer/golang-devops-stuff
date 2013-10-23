package main

import (
	"fmt"
	"net"
)

func (this *Server) Maintenance_Off(conn net.Conn, applicationName string) error {
	err := this.WithPersistentApplication(applicationName, func(app *Application, cfg *Config) error {
		app.Maintenance = false
		return nil
	})
	if err != nil {
		return err
	}
	e := &Executor{NewLogger(NewMessageLogger(conn), "[maintenance:off] ")}
	return this.SyncLoadBalancers(e, []Dyno{}, []Dyno{})
}

func (this *Server) Maintenance_On(conn net.Conn, applicationName string) error {
	err := this.WithPersistentApplication(applicationName, func(app *Application, cfg *Config) error {
		app.Maintenance = true
		return nil
	})
	if err != nil {
		return err
	}
	e := &Executor{NewLogger(NewMessageLogger(conn), "[maintenance:on] ")}
	return this.SyncLoadBalancers(e, []Dyno{}, []Dyno{})
}

func (this *Server) Maintenance_Status(conn net.Conn, applicationName string) error {
	return this.WithApplication(applicationName, func(app *Application, cfg *Config) error {
		Logf(conn, "maintenance: %v\n", app.Maintenance)
		return nil
	})
}

func (this *Server) Maintenance_Url(conn net.Conn, applicationName string, url string) error {
	titleLogger, dimLogger := this.getTitleAndDimLoggers(conn)
	if len(url) == 0 {
		return this.WithApplication(applicationName, func(app *Application, cfg *Config) error {
			fmt.Fprintf(titleLogger, "Getting maintenance page URL for: %v\n", applicationName)
			val, ok := app.Environment["MAINTENANCE_PAGE_URL"]
			if ok {
				fmt.Fprintf(dimLogger, "%v\n", val)
			} else {
				return fmt.Errorf("maintenance page URL configuration key is missing")
			}
			return nil
		})
	} else {
		return this.WithPersistentApplication(applicationName, func(app *Application, cfg *Config) error {
			fmt.Fprintf(titleLogger, "Setting maintenance page URL for: %v\n", applicationName)
			app.Environment["MAINTENANCE_PAGE_URL"] = url
			fmt.Fprintf(dimLogger, "Maintenance page URL is now: %v\n", url)
			return nil
		})
	}
}
