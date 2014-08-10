// Copyright 2014 gandalf authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"github.com/bmizerany/pat"
	"github.com/tsuru/config"
	"github.com/tsuru/gandalf/api"
	"log"
	"net/http"
)

const version = "0.4.1"

func main() {
	dry := flag.Bool("dry", false, "dry-run: does not start the server (for testing purpose)")
	configFile := flag.String("config", "/etc/gandalf.conf", "Gandalf configuration file")
	gVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *gVersion {
		fmt.Printf("gandalf-webserver version %s\n", version)
		return
	}

	err := config.ReadAndWatchConfigFile(*configFile)
	if err != nil {
		msg := `Could not find gandalf config file. Searched on %s.
For an example conf check gandalf/etc/gandalf.conf file.\n %s`
		log.Panicf(msg, *configFile, err)
	}
	router := pat.New()
	router.Post("/user/:name/key", http.HandlerFunc(api.AddKey))
	router.Del("/user/:name/key/:keyname", http.HandlerFunc(api.RemoveKey))
	router.Get("/user/:name/keys", http.HandlerFunc(api.ListKeys))
	router.Post("/user", http.HandlerFunc(api.NewUser))
	router.Del("/user/:name", http.HandlerFunc(api.RemoveUser))
	router.Post("/repository", http.HandlerFunc(api.NewRepository))
	router.Post("/repository/grant", http.HandlerFunc(api.GrantAccess))
	router.Del("/repository/revoke", http.HandlerFunc(api.RevokeAccess))
	router.Del("/repository/:name", http.HandlerFunc(api.RemoveRepository))
	router.Get("/repository/:name", http.HandlerFunc(api.GetRepository))
	router.Put("/repository/:name", http.HandlerFunc(api.RenameRepository))
	router.Get("/repository/:name/archive", http.HandlerFunc(api.GetArchive))
	router.Get("/repository/:name/contents", http.HandlerFunc(api.GetFileContents))
	router.Get("/repository/:name/tree/:path", http.HandlerFunc(api.GetTree))
	router.Get("/repository/:name/tree", http.HandlerFunc(api.GetTree))
	router.Get("/repository/:name/branches", http.HandlerFunc(api.GetBranches))
	router.Get("/repository/:name/tags", http.HandlerFunc(api.GetTags))
	router.Get("/repository/:name/diff/commits", http.HandlerFunc(api.GetDiff))
	router.Get("/healthcheck/", http.HandlerFunc(api.HealthCheck))
	router.Post("/hook/:name", http.HandlerFunc(api.AddHook))

	bind, err := config.GetString("bind")
	if err != nil {
		var perr error
		bind, perr = config.GetString("webserver:port")
		if perr != nil {
			panic(err)
		}
	}
	if !*dry {
		log.Fatal(http.ListenAndServe(bind, router))
	}
}
