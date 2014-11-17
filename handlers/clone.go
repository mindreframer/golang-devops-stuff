package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/therealbill/libredis/client"
	"github.com/therealbill/redskull/common"
	"github.com/zenazn/goji/web"
)

// Clone is not currently exposed as it comes from the first incarnation of the
// idea of a Redis manager. It will likely be incorporated into the pod-level
// handlers and exposed through that route.
func Clone(c web.C, w http.ResponseWriter, r *http.Request) {
	//log.Printf("Someone submitted request to clone from %s to %s", r.Param("originAddress"), r.Param("targetAddress"))
	body, err := ioutil.ReadAll(r.Body)
	var reqdata common.CloneRequest
	err = json.Unmarshal(body, &reqdata)
	if err != nil {
		fmt.Fprint(w, "ohnos unmarshal error")
	}
	log.Printf("%+v", reqdata)
	if err != nil {
		panic("ohnoes")
	}

	originAddress := reqdata.Origin
	cloneAddress := reqdata.Clone
	reconfigureSlaves := reqdata.Reconfig
	promoteClone := reqdata.Promote
	roleRequired := reqdata.Role

	if len(roleRequired) == 0 {
		roleRequired = "master"
	}

	if reconfigureSlaves {
		promoteClone = true
	}

	data := CloneServer(originAddress, cloneAddress, promoteClone, reconfigureSlaves, 3.0, roleRequired)
	fmt.Fprint(w, data)
}

// CloneServer does the heavy lifting to clone one Redis instance to another.
func CloneServer(originHost, cloneHost string, promoteWhenComplete, reconfigureSlaves bool, syncTimeout float64, roleRequired string) (result map[string]string) {
	jobId := uuid.New()
	result = map[string]string{
		"origin":      originHost,
		"clone":       cloneHost,
		"requestTime": fmt.Sprintf("%s", time.Now()),
		"jobid":       jobId,
		"status":      "pending",
		"error":       "",
	}

	if cloneHost == originHost {
		log.Print("Can not clone a host to itself, aborting")
		result["status"] = "ERROR"
		result["error"] = "Can not clone a node to itself"
		return
	}

	// Connect to the Origin node
	originConf := client.DialConfig{Address: originHost}
	origin, err := client.DialWithConfig(&originConf)
	if err != nil {
		log.Println("Unable to connect to origin", err)
		result["status"] = "ERROR"
		result["error"] = "Unable to connect to origin"
		return
	} else {
		log.Print("Connection to origin confirmed")
	}
	// obtain node information
	info, err := origin.Info()
	role := info.Replication.Role
	if err != nil {
		log.Printf("Unable to get the role of the origin instance")
		result["status"] = "ERROR"
		result["error"] = "Unable to get replication role for origin"
		return
	}

	log.Print("Role:", role)
	// verify the role we get matches our condition for a backup
	switch role {
	case roleRequired:
		log.Print("acceptable role confirmed, now to perform a clone...")
	default:
		log.Print("Role mismatch, no clone will be performed")
		result["status"] = "ERROR"
		result["error"] = "Role requirement not met"
		return
	}
	// Now connect to the clone ...
	cloneConf := client.DialConfig{Address: cloneHost}
	clone, err := client.DialWithConfig(&cloneConf)
	if err != nil {
		log.Println("Unable to connect to clone")
		result["status"] = "ERROR"
		result["error"] = "Unable to connect to clone target"
		return
	} else {
		log.Print("Connection to clone confirmed")
	}
	clone.Info()

	oconfig, err := origin.ConfigGet("*")
	if err != nil {
		log.Println("Unable to get origin config, aborting on err:", err)
		result["status"] = "ERROR"
		result["error"] = "Unable to get config from origin"
		return
	}
	// OK, now we are ready to start cloning
	log.Print("Cloning config")
	for k, v := range oconfig {
		// slaveof is not clone-able and is set separately, so skip it
		if k == "slaveof" {
			continue
		}
		err := clone.ConfigSet(k, v)
		if err != nil {
			if !strings.Contains(err.Error(), "Unsupported CONFIG parameter") {
				log.Printf("Unable to set key '%s' to val '%s' on clone due to Error '%s'\n", k, v, err)
			}
		}
	}
	log.Print("Config cloned, now syncing data")
	switch role {
	case "slave":
		// If we are cloning a slave we are assuming it needs to look just like
		// the others, so we simply clone the settings and slave it to the
		// origin's master
		slaveof := strings.Split(oconfig["slaveof"], " ")
		log.Printf("Need to set clone to slave to %s on port %s\n", slaveof[0], slaveof[1])
		slaveres := clone.SlaveOf(slaveof[0], slaveof[1])
		if slaveres != nil {
			log.Printf("Unable to clone slave setting! Error: '%s'\n", slaveres)
		} else {
			log.Print("Successfully cloned new slave")
			return
		}
	case "master":
		// master clones can get tricky.
		// First, slave to the origin nde to get a copy of the data
		log.Print("Role being cloned is 'master'")
		log.Print("First, we need to slave to the original master to pull data down")
		slaveof := strings.Split(originHost, ":")
		slaveres := clone.SlaveOf(slaveof[0], slaveof[1])
		if slaveres != nil {
			if !strings.Contains(slaveres.Error(), "Already connected") {
				log.Printf("Unable to slave clone to origin! Error: '%s'\n", slaveres)
				log.Print("Aborting clone so you can investigate why.")
				return
			}
		}
		log.Printf("Successfully cloned to %s:%s\n", slaveof[0], slaveof[1])

		syncInProgress := true
		new_info, _ := clone.Info()
		syncInProgress = new_info.Replication.MasterSyncInProgress || new_info.Replication.MasterLinkStatus == "down"
		syncTime := 0.0
		if syncInProgress {
			log.Print("Sync in progress...")
			for {
				new_info, _ := clone.Info()
				syncInProgress = new_info.Replication.MasterSyncInProgress || new_info.Replication.MasterLinkStatus == "down"
				if syncInProgress {
					syncTime += .5
					if syncTime >= syncTimeout {
						break
					}
					time.Sleep(time.Duration(500) * time.Millisecond)
				} else {
					break
				}
			}
		}
		if syncInProgress {
			log.Print("Sync took longer than expected, aborting until this is better handled!")
			result["message"] = "Sync in progress"
			return
		}
		// Now we have synced data.
		// Next we need to see if we should promote the new clone to a master
		// this is useful for migrating a master but also for providing a
		// production clone for dev or testing
		log.Print("Now checking for slave promotion")
		if promoteWhenComplete {
			promoted := clone.SlaveOf("no", "one")
			if promoted != nil {
				log.Print("Was unable to promote clone to master, investigate why!")
				return
			}
			log.Print("Promoted clone to master")
			// IF we are migrating a master entirely, we want to reconfigure
			// it's slaves to point to the new master
			// While it might make sense to promote the clone after slaving,
			// doing that means writes are lost in between slave migration and
			// promotion. This gets tricky, which is why by default we don't do it.
			if !reconfigureSlaves {
				log.Print("Not instructed to promote existing slaves")
				log.Print("Clone complete")
				result["status"] = "Complete"
				return
			} else {
				info, _ := origin.Info()
				slaveof := strings.Split(cloneHost, ":")
				desired_port, _ := strconv.Atoi(slaveof[1])
				for index, data := range info.Replication.Slaves {
					log.Printf("Reconfiguring slave %d/%d\n", index, info.Replication.ConnectedSlaves)
					fmt.Printf("Slave data: %+v\n", data)
					slave_connstring := fmt.Sprintf("%s:%d", data.IP, data.Port)
					slaveconn, err := client.DialWithConfig(&client.DialConfig{Address: slave_connstring})
					if err != nil {
						log.Printf("Unable to connect to slave '%s', skipping", slave_connstring)
						continue
					}
					err = slaveconn.SlaveOf(slaveof[0], slaveof[1])
					if err != nil {
						log.Printf("Unable to slave %s to clone. Err: '%s'", slave_connstring, err)
						continue
					}
					time.Sleep(time.Duration(100) * time.Millisecond) // needed to give the slave time to sync.
					slave_info, _ := slaveconn.Info()
					if slave_info.Replication.MasterHost == slaveof[0] {
						if slave_info.Replication.MasterPort == desired_port {
							log.Printf("Slaved %s to clone", slave_connstring)
						} else {
							log.Print("Hmm, slave settings don't match, look into this on slave", data.IP, data.Port)
						}
					}
				}
				result["status"] = "Complete"
			}
		}
		result["status"] = "Complete"
	}
	return
}
