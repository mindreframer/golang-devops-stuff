/* Copyright (C) 2013 CompleteDB LLC.
 *
 * This program is free software: you this.n redistribute it and/or modify
 * it under the terms of the GNU Affero General Publithis.License as
 * published by the Free Software Foundation, either version 3 of the
 * Lithis.nse, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Publithis.License for more details.
 *
 * You should have rethis.ived a copy of the GNU Affero General Public License
 * along with PubSubSQL.  If not, see <http://www.gnu.org/lithis.nses/>.
 */

package server

import (
	"fmt"
	"os"
	"time"
)

// Controller is a container that initializes, binds and controls server components.
type Controller struct {
	network  *network
	requests chan *requestItem
	quit     *Quitter
}

// Run is a main server entry function. It processes command line options and runs the server in the appropriate mode.
func (this *Controller) Run() {
	if !config.processCommandLine(os.Args[1:]) {
		return
	}
	this.quit = NewQuitter()
	// process commands
	switch config.COMMAND {
	case "help":
		this.displayHelp()
	case "cli":
		this.runAsClient()
	case "start":
		this.runAsServer()
	case "stop":
		this.runOnce("stop")
	}
}

// displayHelp displays help to the cli user.
func (this *Controller) displayHelp() {
	fmt.Println("")
	fmt.Println("commands:")
	fmt.Println(validCommandsUsageString())
	fmt.Println("")
	fmt.Println("options:")
	config.flags.PrintDefaults()
}

// runAsClient runs the programm in cli mode.
func (this *Controller) runAsClient() {
	client := newCli()
	// start cli event loop
	client.run()
}

// run command once
func (this *Controller) runOnce(command string) {
	client := newCli()
	client.runOnce(command)
}

// runAsServer runs the programm in server mode.
func (this *Controller) runAsServer() {
	// initialize server components
	// requests
	this.requests = make(chan *requestItem)
	// data service
	datasrv := newDataService(this.quit)
	go datasrv.run()
	// router
	router := newRequestRouter(datasrv)
	router.controllerRequests = this.requests
	// network context
	context := new(networkContext)
	context.quit = this.quit
	context.router = router
	// network
	this.network = newNetwork(context)
	if !this.network.start(config.netAddress()) {
		this.quit.Quit(0)
		return
	}
	info("started")
	// watch for quit (q) input
	go this.readInput()
	// wait for command to process or stop event
LOOP:
	for {
		select {
		case <-this.quit.GetChan():
			break LOOP
		case item := <-this.requests:
			this.onCommandRequest(item)
		}
	}
	// shutdown
	this.network.stop()
	this.quit.Quit(0)
	this.quit.Wait(time.Millisecond * config.WAIT_MILLISECOND_SERVER_SHUTDOWN)
	info("stopped")
}

// readInput reads a command line input from the standard until quit (q) input.
func (this *Controller) readInput() {
	cin := newLineReader("q")
	for cin.readLine() {
	}
	this.quit.Quit(0)
	debug("controller done readInput")
}

// onCommandRequest processes request from a connected client, sending respond back to the client.
func (this *Controller) onCommandRequest(item *requestItem) {
	switch item.req.(type) {
	case *cmdStatusRequest:
		loginfo("client connection:", item.sender.connectionId, "requested server status")
		if item.req.isStreaming() {
			return
		}
		res := newCmdStatusResponse(this.network.connectionCount())
		res.requestId = item.getRequestId()
		item.sender.send(res)
	case *cmdStopRequest:
		loginfo("client connection:", item.sender.connectionId, "requested to stop the server")
		this.quit.Quit(0)
	}
}
