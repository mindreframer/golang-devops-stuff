package main

import (
	"fmt"
	"net"
	"strings"

	. "github.com/Sendhub/logserver"
	"github.com/Sendhub/logserver/server"
)

var (
	activeDrains = []*server.Drainer{}
)

func initDrains(this *Server) {
	this.WithConfig(func(cfg *Config) error {
		for _, app := range cfg.Applications {
			for _, address := range app.Drains {
				drain := this.LogServer.StartDrainer(address, EntryFilter{
					Application: app.Name,
				})
				activeDrains = append(activeDrains, drain)
			}
		}
		return nil
	})
}

func (this *Server) Drains_Add(conn net.Conn, applicationName string, addresses []string) error {
	return this.WithPersistentApplication(applicationName, func(app *Application, cfg *Config) error {
		app.Drains = this.UniqueStringsAppender(conn, app.Drains, addresses, "drain",
			func(addItem string) {
				// Open a new drain.
				drain := this.LogServer.StartDrainer(addItem, EntryFilter{Application: applicationName})
				activeDrains = append(activeDrains, drain)
			},
		)
		return nil
	})
}

func (this *Server) Drains_List(conn net.Conn, applicationName string) error {
	titleLogger, dimLogger := this.getTitleAndDimLoggers(conn)

	fmt.Fprintf(titleLogger, "=== Listing drains for %v\n", applicationName)

	return this.WithApplication(applicationName, func(app *Application, cfg *Config) error {
		for _, address := range app.Drains {
			fmt.Fprintf(dimLogger, "%v\n", address)
		}
		return nil
	})
}

func (this *Server) Drains_Remove(conn net.Conn, applicationName string, addresses []string) error {
	err := this.WithPersistentApplication(applicationName, func(app *Application, cfg *Config) error {
		app.Drains = this.UniqueStringsRemover(conn, app.Drains, addresses, "drain", nil)
		return nil
	})
	if err != nil {
		return err
	}

	// Close and remove any matching active drains.
	newDrains := []*server.Drainer{}
	for _, drain := range activeDrains {
		keep := true
		if drain.Filter.Application == applicationName {
			for _, removeAddress := range addresses {
				if strings.ToLower(removeAddress) == strings.ToLower(drain.Address) {
					keep = false
				}
			}
		}
		if !keep {
			drain.Close()
		} else {
			newDrains = append(newDrains, drain)
		}
	}
	activeDrains = newDrains
	return err
}
