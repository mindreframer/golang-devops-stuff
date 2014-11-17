package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/therealbill/airbrake-go"
	"github.com/therealbill/libredis/client"
	"github.com/therealbill/redskull/common"
	"github.com/zenazn/goji/web"
)

// WEB UI CALL
func RebalanceHTML(c web.C, w http.ResponseWriter, r *http.Request) {
	context := PageContext{Title: "Rebalance Attempt Complete", ViewTemplate: "rebalance_complete", Constellation: ManagedConstellation}
	ManagedConstellation.Balance()
	context.Refresh = true
	context.RefreshTime = 30
	context.RefreshURL = "/constellation/"
	render(w, context)
}

func ConstellationInfoHTML(c web.C, w http.ResponseWriter, r *http.Request) {
	title := "Constellation Information"
	subtitle := ManagedConstellation.Name
	log.Printf("URL: '%s'", r.URL)
	context := PageContext{Title: title, SubTitle: subtitle, ViewTemplate: "show_constellation", Constellation: ManagedConstellation, Data: ManagedConstellation}
	render(w, context)

}

// API Calls

func RebalanceJSON(c web.C, w http.ResponseWriter, r *http.Request) {
	ManagedConstellation.Balance()
	response := InfoResponse{Status: "COMPLETE", StatusMessage: "Rebalance attempt completed", Data: ManagedConstellation.IsBalanced()}
	packed, _ := json.Marshal(response)
	w.Write(packed)
}

func ConstellationInfoJSON(c web.C, w http.ResponseWriter, r *http.Request) {
	packed, _ := json.Marshal(ManagedConstellation)
	w.Write(packed)
}

func DoFailoverJSON(ctx web.C, w http.ResponseWriter, r *http.Request) (err error) {
	podname := ctx.URLParams["podName"]
	didFailover, err := ManagedConstellation.Failover(podname)
	if err != nil {
		retcode, emsg := handleFailoverError(podname, r, err)
		log.Printf("%d: '%s'", retcode, emsg)
		http.Error(w, emsg, retcode)
		return
	}
	if !didFailover {
		retcode, emsg := handleFailoverError(podname, r, err)
		log.Printf("%d: '%s'", retcode, emsg)
		http.Error(w, emsg, retcode)
		return
	}
	return nil
}

func APIFailover(c web.C, w http.ResponseWriter, r *http.Request) {
	var (
		response InfoResponse
		reqdata  common.FailoverRequest
	)
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
	reqdata.Podname = c.URLParams["podName"]
	ok, err := ManagedConstellation.Failover(reqdata.Podname)
	if err != nil {
		em := err.Error()
		em = strings.TrimSpace(em)
		response.Status = "ERROR"
		switch em {
		case "NOGOODSLAVE No suitable slave to promote":
			response.Status = "NOGOODSLAVE"
			response.StatusMessage = "No suitable slave to promote"
		default:
			log.Printf("'%s'", em)
			response.StatusMessage = err.Error()
		}
		packed, _ := json.Marshal(response)
		w.Write(packed)
		return
	}
	if !ok {
		response.Status = "NOTOK"
		response.StatusMessage = "Unknown issue during failover request submission"
		packed, _ := json.Marshal(response)
		w.Write(packed)
	}
	response.Status = "SUCCESS"
	response.StatusMessage = "Failover command accepted"
	if reqdata.ReturnNew {
		newmaster, err := ManagedConstellation.GetMaster(reqdata.Podname)
		if err != nil {
			response.Status = "ERROR"
			response.StatusMessage = err.Error()
		}
		response.Data = newmaster
	}
	packed, _ := json.Marshal(response)
	w.Write(packed)
}

func APIGetSlaves(c web.C, w http.ResponseWriter, r *http.Request) {
	var response InfoResponse
	podName := c.URLParams["podName"]
	slaves, err := ManagedConstellation.GetSlaves(podName)
	response.Data = slaves
	if err != nil {
		response.Status = "ERROR"
		response.StatusMessage = err.Error()
	}
	packed, _ := json.Marshal(response)
	w.Write(packed)
}

func APIGetMaster(c web.C, w http.ResponseWriter, r *http.Request) {
	var response InfoResponse
	podName := c.URLParams["podName"]
	master, err := ManagedConstellation.GetMaster(podName)
	if err != nil {
		em := fmt.Errorf("Sentinel command error '%s'", err)
		e := airbrake.ExtendedNotification{ErrorClass: "Sentinel.Command", Error: em}
		err = airbrake.ExtendedError(e, r)
		if err != nil {
			log.Print("airbrake error:", err)
		}
		response.Status = "COMMANDERROR"
		response.StatusMessage = err.Error()
	} else {
		var addr client.MasterAddress
		addr = master
		response.Data = addr
		if len(addr.Host) == 0 {
			response.StatusMessage = "No master found"
			response.Status = "NOTFOUND"
			w.WriteHeader(404)
			return
		} else {
			response.Status = "SUCCESS"
		}
	}
	packed, _ := json.Marshal(response)
	w.Write(packed)
}

func APIMonitorPod(c web.C, w http.ResponseWriter, r *http.Request) {
	var (
		response InfoResponse
		reqdata  common.MonitorRequest
	)
	podName := c.URLParams["podName"]
	body, err := ioutil.ReadAll(r.Body)
	err = json.Unmarshal(body, &reqdata)
	if err != nil {
		retcode, em := throwJSONParseError(r)
		log.Print(em)
		http.Error(w, em, retcode)
	}
	reqdata.Podname = podName
	ok, err := ManagedConstellation.MonitorPod(podName, reqdata.MasterAddress, reqdata.MasterPort, reqdata.Quorum, reqdata.AuthToken)

	if !ok {
		response.StatusMessage = fmt.Sprintf("Pod '%s' failed to reach sentinel quorum.", reqdata.Podname)
		response.Status = "INCOMPLETE"
		em := fmt.Errorf("MONITOR pod '%s' failed to reach quorum", reqdata.Podname)
		e := airbrake.ExtendedNotification{ErrorClass: "Pod.Quorum", Error: em}
		err = airbrake.ExtendedError(e, r)
		if err != nil {
			log.Print("airbrake error:", err)
		}
	} else {
		response.Status = "COMPLETE"
	}

	packed, _ := json.Marshal(response)
	w.Write(packed)
}

func APIRemovePod(c web.C, w http.ResponseWriter, r *http.Request) {
	var (
		response InfoResponse
	)
	podName := c.URLParams["podName"]
	log.Print("Removing pod:", podName)

	_, err := ManagedConstellation.RemovePod(podName)
	if err != nil {
		response.Status = "COMMANDERROR"
		response.StatusMessage = err.Error()
	} else {
		response.Status = "COMPLETE"
		response.StatusMessage = fmt.Sprintf("Not monitoring pod '%s'", podName)
	}
	packed, err := json.Marshal(response)
	if err != nil {
		log.Print("Unable to pack JSON, err:", err)
	}
	w.Write(packed)
}

func APIGetPodMap(c web.C, w http.ResponseWriter, r *http.Request) {
	var (
		response InfoResponse
	)
	pods, _ := ManagedConstellation.GetPodMap()
	response.Status = "COMPLETE"
	response.Data = pods
	packed, err := json.Marshal(response)
	if err != nil {
		log.Print("Unable to pack JSON, err:", err)
	}
	w.Write(packed)
}

func APIGetPods(c web.C, w http.ResponseWriter, r *http.Request) {
	var (
		response InfoResponse
	)
	pods := ManagedConstellation.GetPods()
	response.Status = "COMPLETE"
	response.Data = pods
	packed, err := json.Marshal(response)
	if err != nil {
		log.Print("Unable to pack JSON, err:", err)
	}
	w.Write(packed)
}

func APIGetPod(c web.C, w http.ResponseWriter, r *http.Request) {
	var (
		response InfoResponse
	)
	podname := c.URLParams["podName"]
	log.Print("pulling API for pod " + podname)
	if podname == "" {
		err := fmt.Errorf("API:GP called w/o a pod??")
		log.Print("API:GP Error:", err)
		response.Status = "ERROR"
		response.StatusMessage = err.Error()
	} else {
		pod, err := ManagedConstellation.GetPod(podname)
		if pod.Name > "" {
			response.Status = "COMPLETE"
			log.Printf("Pod data: %+v", pod)
			response.Data = pod
		} else {
			log.Print("API:GP Error:", err)
			response.Status = "ERROR"
			response.StatusMessage = err.Error()
			response.Data = pod
		}
	}
	packed, err := json.Marshal(response)
	if err != nil {
		log.Print("Unable to pack JSON, err:", err)
	}
	log.Print(packed)
	w.Write(packed)
}
