/*
Copyright 2011 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	port  = flag.Int("port", 8000, "port")
	crash = flag.Bool("crash", false, "crash on start")
)

func crashHandler(w http.ResponseWriter, r *http.Request) {
	status := 2
	if st := r.FormValue("status"); st != "" {
		status, _ = strconv.Atoi(st)
	}
	fmt.Fprintf(os.Stderr, "crashing with status %d", status)
	os.Exit(status)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "pid=%d\n", os.Getpid())
	cwd, _ := os.Getwd()
	fmt.Fprintf(w, "cwd=%s\n", cwd)
	fmt.Fprintf(w, "uid=%d\n", os.Getuid())
	fmt.Fprintf(w, "euid=%d\n", os.Geteuid())
	fmt.Fprintf(w, "gid=%d\n", os.Getgid())

	groups, gerr := exec.Command("groups").CombinedOutput()
	if gerr != nil {
		fmt.Fprintf(w, "groups_err=%q\n", gerr)
	} else {
		fmt.Fprintf(w, "groups=%s\n", strings.TrimSpace(string(groups)))
	}

	ulimitN, _ := exec.Command("ulimit", "-n").Output()
	fmt.Fprintf(w, "ulimit_nofiles=%s\n", strings.TrimSpace(string(ulimitN)))

	env := os.Environ()
	sort.Strings(env)
	for _, env := range env {
		fmt.Fprintf(w, "%s\n", env)
	}
}

func logNoise() {
	for {
		log.Printf("some log noise")
		time.Sleep(1 * time.Second)
	}
}

func main() {
	flag.Parse()

	if *crash {
		log.Fatalf("fake crash on start")
	}

	cmd := exec.Command("/usr/bin/perl", "-e", `while(1) { print time(), "\n"; sleep 1; }`)
	if err := cmd.Start(); err != nil {
		log.Fatalf("error running child: %v", err)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("error listening on port %d: %v", *port, err)
	}

	fmt.Fprintf(os.Stdout, "Hello on stdout; listening on port %d\n", *port)
	fmt.Fprintf(os.Stderr, "Hello on stderr\n")
	go logNoise()

	http.HandleFunc("/crash", crashHandler)
	http.HandleFunc("/", statusHandler)

	s := &http.Server{}
	err = s.Serve(ln)
	log.Printf("Serve: %v", err)
	if err != nil {
		os.Exit(1)
	}
}
