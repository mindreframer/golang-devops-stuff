package main

import (
	"fmt"
	"net"
	"strings"
)

func (this *Server) Domains_Add(conn net.Conn, applicationName string, domains []string) error {
	titleLogger, dimLogger := this.getTitleAndDimLoggers(conn)
	fmt.Fprintf(titleLogger, "=== Adding domains to %v\n", applicationName)

	err := this.WithPersistentApplication(applicationName, func(app *Application, cfg *Config) error {
		fmt.Fprintf(dimLogger, "new=%v\n", domains)
		for _, domain := range domains {
			if len(domain) > 0 {
				foundAlready := false
				for _, existing := range app.Domains {
					if strings.ToLower(existing) == strings.ToLower(domain) {
						foundAlready = true
						fmt.Fprintf(dimLogger, "    Domain already added: %v\n", domain)
						break
					}
				}
				// Check to make sure the domain doesn't already exist in another app.
				for _, otherApp := range cfg.Applications {
					if otherApp.Name != app.Name {
						for _, existing := range otherApp.Domains {
							if strings.ToLower(existing) == strings.ToLower(domain) {
								foundAlready = true
								fmt.Fprintf(dimLogger, "    Domain already in-use by another application: %v\n", domain)
								break
							}
						}
					}
				}
				if !foundAlready {
					fmt.Fprintf(dimLogger, "    Adding domain: %v\n", domain)
					app.Domains = append(app.Domains, domain)
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	e := &Executor{dimLogger}
	return this.SyncLoadBalancers(e, []Dyno{}, []Dyno{})
}

func (this *Server) Domains_List(conn net.Conn, applicationName string) error {
	titleLogger, dimLogger := this.getTitleAndDimLoggers(conn)
	fmt.Fprintf(titleLogger, "=== Domains for %v\n", applicationName)

	return this.WithApplication(applicationName, func(app *Application, cfg *Config) error {
		for _, domain := range app.Domains {
			fmt.Fprintf(dimLogger, "%v\n", domain)
		}
		return nil
	})
}

func (this *Server) Domains_Remove(conn net.Conn, applicationName string, domains []string) error {
	titleLogger, dimLogger := this.getTitleAndDimLoggers(conn)
	fmt.Fprintf(titleLogger, "=== Removing domains from %v\n", applicationName)

	err := this.WithPersistentApplication(applicationName, func(app *Application, cfg *Config) error {
		nDomains := []string{}
		for _, existing := range app.Domains {
			removalRequested := false
			for _, remove := range domains {
				if remove == existing {
					removalRequested = true
					break
				}
			}
			if !removalRequested {
				nDomains = append(nDomains, existing)
			} else {
				fmt.Fprintf(dimLogger, "    Removing domain: %v\n", existing)
			}
		}
		app.Domains = nDomains
		return nil
	})
	if err != nil {
		return err
	}
	e := &Executor{dimLogger}
	return this.SyncLoadBalancers(e, []Dyno{}, []Dyno{})
}
