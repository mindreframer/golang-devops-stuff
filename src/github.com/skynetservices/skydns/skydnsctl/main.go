// Copyright (c) 2013 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/skynetservices/skydns/client"
	"github.com/skynetservices/skydns/msg"
	"os"
	"strconv"
	"strings"
)

func writeError(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(1)
}

func writeService(c *cli.Context, service *msg.Service) {
	if c.GlobalBool("json") {
		if err := json.NewEncoder(os.Stdout).Encode(service); err != nil {
			writeError(err)
		}
	} else {
		fmt.Printf("UUID: %s\nName: %s\nHost: %s\nPort: %d\nEnvironment: %s\nRegion: %s\nVersion: %s\n\n",
			service.UUID,
			service.Name,
			service.Host,
			service.Port,
			service.Environment,
			service.Region,
			service.Version)

		fmt.Printf("TTL %d\nRemaining TTL: %d\n",
			service.TTL,
			service.RemainingTTL())
	}
}

func newClientFromContext(c *cli.Context) (*client.Client, error) {
	var (
		base   = c.GlobalString("host")
		dns    = c.GlobalString("dns")
		domain = c.GlobalString("domain")
		secret = c.GlobalString("secret")
	)
	s, e := client.NewClient(base, secret, domain, dns)
	if e == nil {
		s.DNS = c.Bool("d") // currently only defined when listing services
	}
	return s, e
}

func loadCommands(app *cli.App) {
	// default to listing a service
	app.Action = getAction

	app.Flags = []cli.Flag{
		cli.BoolFlag{"json", "output to json"},
		cli.StringFlag{"host", os.Getenv("SKYDNS"), "url to SkyDNS's HTTP endpoints (defaults to env. var. SKYDNS)"},
		cli.StringFlag{"dns",
			func() string {
				if x := os.Getenv("SKYDNS_DNS"); x != "" {
					if strings.HasPrefix(x, "http") {
						return x
					}
					return "http://" + x // default to http for now
				}
				return "127.0.0.1:53"
			}(), "DNS port of SkyDNS's DNS endpoint (defaults to env. var. SKYDNS_DNS)"},
		cli.StringFlag{"domain",
			func() string {
				if x := os.Getenv("SKYDNS_DOMAIN"); x != "" {
					return x
				}
				return "skydns.local"
			}(), "DNS domain of SkyDNS (defaults to env. var. SKYDNS_DOMAIN))"},
		cli.StringFlag{"secret", "", "secret to authenticate with"},
	}

	app.Commands = []cli.Command{
		{
			Name:   "list",
			Usage:  "list a service from skydns",
			Action: getAction,
			Flags:  []cli.Flag{cli.BoolFlag{"d", "use DNS instead of HTTP"}},
		},
		{
			Name:   "add",
			Usage:  "add a new service to skydns",
			Action: addAction,
		},
		{
			Name:   "delete",
			Usage:  "delete a service from skydns",
			Action: deleteAction,
		},
		{
			Name:   "update",
			Usage:  "update a service's ttl in skydns",
			Action: updateAction,
		},
	}
}

// Add a new service to skydns
//
// format: skydnsctl  1001 '{"Name":"TestService","Version":"1.0.0","Environment":"Production","Region":"Test","Host":"web1.site.com","Port":9000,"TTL":10}'
func addAction(c *cli.Context) {
	skydns, err := newClientFromContext(c)
	if err != nil {
		writeError(err)
	}

	var (
		service *msg.Service
		uuid    = c.Args().Get(0)
		rawData = c.Args().Get(1)
	)

	if err := json.Unmarshal([]byte(rawData), &service); err != nil {
		writeError(err)
	}

	if err := skydns.Add(uuid, service); err != nil {
		writeError(err)
	}
	fmt.Printf("%s added to skydns\n", uuid)
}

// Remove an existing service from skydns
//
// format: skydnsctl delete 1001
func deleteAction(c *cli.Context) {
	skydns, err := newClientFromContext(c)
	if err != nil {
		writeError(err)
	}

	uuid := c.Args().Get(0)

	if err := skydns.Delete(uuid); err != nil {
		writeError(err)
	}
	fmt.Printf("%s removed from skydns\n", uuid)
}

// Update an existing service in skydns
//
// format: skydnsctl update  1001 10
func updateAction(c *cli.Context) {
	skydns, err := newClientFromContext(c)
	if err != nil {
		writeError(err)
	}

	var (
		uuid   = c.Args().Get(0)
		rawTtl = c.Args().Get(1)
	)

	ttl, err := strconv.Atoi(rawTtl)
	if err != nil {
		writeError(err)
	}

	if err := skydns.Update(uuid, uint32(ttl)); err != nil {
		writeError(err)
	}
	fmt.Printf("%s ttl updated to %d\n", uuid, ttl)
}

// Get a existing service or list all services in skydns
//
// format: skydnsctl || skydnsctl 1001
func getAction(c *cli.Context) {
	skydns, err := newClientFromContext(c)
	if err != nil {
		writeError(err)
	}

	// Get a specific service
	if uuid := c.Args().Get(0); uuid != "" {
		service, err := skydns.Get(uuid)
		if err != nil {
			writeError(err)
		}

		writeService(c, service)
	} else { // or get all services
		var services []*msg.Service
		var err error
		if skydns.DNS {
			services, err = skydns.GetAllServicesDNS()
		} else {
			services, err = skydns.GetAllServices()
		}
		if err != nil {
			writeError(err)
		}

		for _, service := range services {
			writeService(c, service)
			fmt.Printf("\n----\n")
		}
	}
}

func main() {
	app := cli.NewApp()
	app.Author = "skydns"
	app.Name = "skydnsctl"
	app.Version = "0.2"

	loadCommands(app)

	if err := app.Run(os.Args); err != nil {
		writeError(err)
	}
}
