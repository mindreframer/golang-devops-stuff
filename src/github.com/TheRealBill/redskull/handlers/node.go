package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/therealbill/libredis/client"
	"github.com/zenazn/goji/web"
)

// ShowNodes shows the node listing page
func ShowNodes(c web.C, w http.ResponseWriter, r *http.Request) {
	context := NewPageContext()
	//NodeMaster.LoadNodes()
	context.Data = context.Constellation.NodeMap
	context.Title = "Red Skull: Known Nodes"
	context.ViewTemplate = "show-nodes"
	render(w, context)
}

// ShowNode handles the individual node display
func ShowNode(c web.C, w http.ResponseWriter, r *http.Request) {
	target := c.URLParams["name"]
	title := fmt.Sprintf("Node: %s", target)
	context := NewPageContext()
	context.Title = title
	context.ViewTemplate = "show-node"
	podname := context.Constellation.NodeNameToPodMap[target]
	log.Printf("Getting node for pod: %s", podname)
	node, _ := context.Constellation.GetNode(target, podname, "")
	context.Node = node
	render(w, context)
}

// AddNode isn't currently used but would add a non-master, non-slave node.
// This will be refactored out to a provisioning layer
func AddNode(c web.C, w http.ResponseWriter, r *http.Request) {
	target := c.URLParams["nodeName"]
	node := NodeMaster.GetNode(target)
	title := fmt.Sprintf("Add Node %s", target)
	context := PageContext{Title: title, ViewTemplate: "add-node-form", NodeMaster: NodeMaster, Data: node}
	render(w, context)
}

// GetNodeJSON returns the JSON output of the data known about a node
func GetNodeJSON(c web.C, w http.ResponseWriter, r *http.Request) {
	target := c.URLParams["name"]
	//node := NodeMaster.GetNode(target)
	context := NewPageContext()
	podname := context.Constellation.NodeNameToPodMap[target]
	log.Printf("Getting node for pod: %s", podname)
	node, _ := context.Constellation.GetNode(target, podname, "")
	response := InfoResponse{Status: "COMPLETE", StatusMessage: "Pod Info Retrieved", Data: node}
	log.Printf("[%s]: %+v", target, node)
	packed, _ := json.Marshal(response)
	w.Write(packed)
}

// AddNodeHTMLProcessor is the target for the AddNode form's action
func AddNodeHTMLProcessor(c web.C, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	// initially this will require a node to already be available on the network
	// will also need to add auth for node. Free nodes' auth string common
	// among the nodes on bootup, but then changed to a constellation-specific
	// one. This will help ensure only the constellation which manages the
	// instance can do so
	//
	// Once I've got a Redis Anywhere API built/figured out it will call that
	// to provision the node then add it into this system
	nodename := c.URLParams["nodeName"]
	node := NodeMaster.GetNode(nodename)
	context := NewPageContext()
	context.Title = "Node Node Result"
	context.ViewTemplate = "slave-added"
	context.NodeMaster = NodeMaster
	address := r.FormValue("host")
	portstr := r.FormValue("port")
	port, _ := strconv.Atoi(portstr)
	log.Printf("Name: %s. Address: %s, Port: %d", nodename, address, port)
	_ = node
	type results struct {
		Name     string
		Address  string
		Port     int
		Error    string
		HasError bool
		NodeURL  string
	}
	res := results{Name: nodename, Address: address, Port: port}
	nodeconn, err := client.Dial(address, port)
	if err != nil {
		log.Print("ERR: Dialing node -", err)
		//context.Data = err
		render(w, context)
		return
	}
	defer nodeconn.ClosePool()
	_ = nodeconn
	context.Data = res // wtf, why is this insisting it needs to be a comparison?!
	render(w, context)

}
