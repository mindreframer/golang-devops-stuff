package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"encoding/json"
	"github.com/therealbill/redskull/actions"
	"github.com/therealbill/redskull/common"
	"github.com/zenazn/goji/web"
	"io/ioutil"
)

// AddPodHTML is the action target for adding a pod. It does the heavy lifting
func AddPodHTML(c web.C, w http.ResponseWriter, r *http.Request) {
	// Change to use actions package
	log.Print("########### ADD POD FORM PROCESSING ###########")
	r.ParseForm()
	log.Print("add pod post called")
	context := NewPageContext()
	context.Title = "Pod Add Result"
	context.ViewTemplate = "podaddpost"

	podname := r.FormValue("podname")
	address := r.FormValue("iphost")
	auth := r.FormValue("authtoken")
	addpair := strings.Split(address, ":")
	host := addpair[0]
	port, err := strconv.Atoi(addpair[1])
	quorum, _ := strconv.Atoi(r.FormValue("quorum"))
	log.Printf("Name: %s. Address: %s, Quorum: %d", podname, address, quorum)
	type results struct {
		Name     string
		Address  string
		Quorum   int
		Error    string
		HasError bool
		PodURL   string
		Pod      actions.RedisPod
	}
	res := results{Name: podname, Address: address, Quorum: quorum}
	_, err = ManagedConstellation.MonitorPod(podname, host, port, quorum, auth)
	if err != nil {
		log.Printf("Error on addpod: %s", err.Error())
		res.Error = err.Error()
		res.HasError = true
		context.Data = res
		render(w, context)
		return
	}
	// I hate this, jsut here for debugging
	time.Sleep(25 * time.Millisecond)
	pod, err := ManagedConstellation.GetPod(podname)
	if err != nil {
		log.Printf("H:MP-> Unable to get newly added pod! Error: %s", err.Error())
		res.Error = err.Error()
		res.HasError = true
	}
	if len(pod.Master.Name) == 0 {
		res.Error = "H:MP-> Unable to get newly added pod!"
		res.HasError = true
	}
	//log.Printf("H:AP-> got pod with master node = %+v", pod.Master)
	if len(pod.Master.Name) > 0 {
		ManagedConstellation.PodMap[podname] = pod
		//context.NodeMaster.AddNode(pod.Master)
	}
	context.Pod = pod
	context.Data = res
	log.Print("########### ADD POD FORM PROCESSED ###########")
	render(w, context)
}

// AddPodForm displays the form for adding a pod
func AddPodForm(c web.C, w http.ResponseWriter, r *http.Request) {
	context := PageContext{Title: "Add Pod to Constellation", ViewTemplate: "addpod", Constellation: ManagedConstellation}
	render(w, context)
}

// AddSentinelForm displays the form for adding a sentinel
func AddSentinelForm(c web.C, w http.ResponseWriter, r *http.Request) {
	context := PageContext{Title: "Add Sentinel to Constellation", ViewTemplate: "addsentinel", Constellation: ManagedConstellation}
	render(w, context)
}

// AddSentinelHTML does the heavy lifting of adding a sentinel. It is the
// action target for the sentinel add form
func AddSentinelHTML(c web.C, w http.ResponseWriter, r *http.Request) {
	// Change to use actions package
	log.Print("########### ADD SENTINEL FORM PROCESSING ###########")
	r.ParseForm()
	context := NewPageContext()
	context.Title = "Sentinel Add Result"
	context.ViewTemplate = "sentineladdpost"
	context.Refresh = true
	context.RefreshTime = 2
	context.RefreshURL = "/constellation/"

	name := r.FormValue("name")
	address := r.FormValue("iphost")
	type results struct {
		Name     string
		Address  string
		Error    string
		HasError bool
	}
	res := results{Name: name, Address: address}
	err := ManagedConstellation.AddSentinelByAddress(address)
	if err != nil {
		log.Printf("Error on addsentinel: %s", err.Error())
		res.Error = err.Error()
		res.HasError = true
	}
	sentinel, ok := ManagedConstellation.RemoteSentinels[address]
	if len(sentinel.Name) == 0 || !ok {
		res.Error = "H:MP-> Unable to get newly added sentinel!"
		res.HasError = true
	}
	context.Data = res
	log.Print("########### ADD SENTINEL FORM PROCESSED ###########")
	render(w, context)
}

func AddPodJSON(c web.C, w http.ResponseWriter, r *http.Request) {
	// Change to use actions package
	var reqdata common.MonitorRequest
	body, err := ioutil.ReadAll(r.Body)
	err = json.Unmarshal(body, &reqdata)
	if err != nil {
		retcode, em := throwJSONParseError(r)
		log.Println(body)
		if retcode >= 400 {
			http.Error(w, em, retcode)
			return
		}
	}

	type results struct {
		Name     string
		Address  string
		Port     int
		Quorum   int
		Error    string
		HasError bool
		PodURL   string
	}
	res := results{Name: reqdata.Podname, Address: reqdata.MasterAddress, Port: reqdata.MasterPort, Quorum: reqdata.Quorum}
	_, err = ManagedConstellation.MonitorPod(reqdata.Podname, reqdata.MasterAddress, reqdata.MasterPort, reqdata.Quorum, reqdata.AuthToken)
	if err != nil {
		res.Error = err.Error()
		res.HasError = true
	}
	packed, _ := json.Marshal(res)
	w.Write(packed)
}
