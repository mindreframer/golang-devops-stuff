package actions

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/groupcache"
	"github.com/therealbill/libredis/client"
)

const GCPORT = "8008"

// SentinelPodConfig is a struct carrying information about a Pod's config as
// pulled from the sentinel config file.
type SentinelPodConfig struct {
	IP        string
	Port      int
	Quorum    int
	Name      string
	AuthToken string
	Sentinels map[string]string
}

// LocalSentinelConfig is a struct holding information about the sentinel RS is
// running on.
type LocalSentinelConfig struct {
	Name              string
	Host              string
	Port              int
	ManagedPodConfigs map[string]SentinelPodConfig
	Dir               string
}

// Constellation is a construct which holds information about the constellation
// as well providing an interface for taking actions against it.
type Constellation struct {
	Name                string
	PodMap              map[string]*RedisPod
	LocalPodMap         map[string]*RedisPod
	RemotePodMap        map[string]*RedisPod
	PodsInError         []*RedisPod
	Connected           bool
	RemoteSentinels     map[string]*Sentinel
	BadSentinels        map[string]*Sentinel
	LocalSentinel       Sentinel
	SentinelConfigName  string
	SentinelConfig      LocalSentinelConfig
	PodToSentinelsMap   map[string][]*Sentinel
	Balanced            bool
	AuthCache           *PodAuthCache
	Groupname           string
	Peers               *groupcache.HTTPPool
	PeerList            map[string]string
	PodAuthMap          map[string]string
	NodeMap             map[string]*RedisNode
	NodeNameToPodMap    map[string]string
	ConfiguredSentinels map[string]interface{}
	Metrics             ConstellationStats
	LocalOverrides      SentinelOverrides
}
type SentinelOverrides struct {
	BindAddress string
}

// GetConstellation returns an instance of a constellation. It requires the
// configuration and a group name. The group name identifies the cluster the
// constellation, and hence this RedSkull instance, belongs to.
// In the future this will be used in clsuter coordination as well as for a
// protective measure against cluster merge
func GetConstellation(name, cfg, group, sentinelAddress string) (*Constellation, error) {
	con := Constellation{Name: name}
	con.SentinelConfig.ManagedPodConfigs = make(map[string]SentinelPodConfig)
	con.PodToSentinelsMap = make(map[string][]*Sentinel)
	con.RemoteSentinels = make(map[string]*Sentinel)
	con.BadSentinels = make(map[string]*Sentinel)
	con.PodAuthMap = make(map[string]string)
	con.LocalPodMap = make(map[string]*RedisPod)
	con.RemotePodMap = make(map[string]*RedisPod)
	con.NodeMap = make(map[string]*RedisNode)
	con.PeerList = make(map[string]string)
	con.NodeNameToPodMap = make(map[string]string)
	con.ConfiguredSentinels = make(map[string]interface{})
	con.Groupname = group
	con.LocalOverrides = SentinelOverrides{BindAddress: sentinelAddress}
	con.SentinelConfigName = cfg
	con.LoadSentinelConfigFile()
	con.LoadLocalPods()
	con.LoadRemoteSentinels()
	con.Balanced = true
	con.PeerList = make(map[string]string)
	log.Printf("Metrics: %+v", con.GetStats())
	return &con, nil
}

// ConstellationStats holds mtrics about the constellation. As the
// Constellation term is undergoing a change, this will also need to change to
// reflect the new terminalogy.
// As soon as it is determined
type ConstellationStats struct {
	PodCount        int
	NodeCount       int
	TotalPodMemory  int64
	TotalNodeMemory int64
	SentinelCount   int
	PodSizes        map[int64]int64
	MemoryUsed      int64
	MemoryPctAvail  float64
}

// GetStats returns metrics about the constellation
func (c *Constellation) GetStats() ConstellationStats {
	// first: pod crawling
	var metrics ConstellationStats
	metrics.PodSizes = make(map[int64]int64)
	pmap, _ := c.GetPodMap()
	metrics.PodCount = len(pmap)
	metrics.SentinelCount = len(c.RemoteSentinels) + 1

	for _, pod := range pmap {
		master := pod.Master
		if master == nil {
			address := fmt.Sprintf("%s:%d", pod.Info.IP, pod.Info.Port)
			var err error
			master, err = c.GetNode(address, pod.Name, pod.AuthToken)
			if err != nil {
				log.Printf("Unable to get master for pod '%s', ERR='%s'", pod.Name, err)
				continue
			}
			metrics.NodeCount++
			for _, slave := range master.Slaves {
				metrics.NodeCount++
				metrics.TotalNodeMemory += int64(slave.MaxMemory)
			}
		}
		podmem := int64(master.MaxMemory)
		metrics.TotalPodMemory += podmem
		metrics.TotalNodeMemory += int64(podmem)
		metrics.PodSizes[podmem]++
	}
	log.Printf("Metrics: %+v", metrics)
	c.Metrics = metrics
	return metrics
}

// GetNode will retun an instance of a RedisNode.
// It also attempts to determine dynamic data such as sentinels and booleans
// like CanFailover
func (c *Constellation) GetNode(name, podname, auth string) (node *RedisNode, err error) {
	node, exists := c.NodeMap[name]
	if exists {
		if node.LastUpdateValid {
			node.UpdateData()
			c.NodeMap[name] = node
			return node, nil
		}
		didUpdate, err := node.UpdateData()
		if err != nil {
			log.Print("ERROR in GetNode:Update -> ", err)
			return node, err
		}
		if !didUpdate {
			log.Print("Update not needed, data still fresh")
		}
		c.NodeMap[name] = node
		c.NodeNameToPodMap[name] = podname
		return node, err
	}
	if auth == "" {
		log.Print("Auth was blank when called, trying to determine it from authcache - ", podname)
		auth = c.GetPodAuth(podname)
	}
	host, port, err := GetAddressPair(name)
	if err != nil {
		log.Print("Unable to determine connection info. Err:", err)
		return
	}
	dnode, err := LoadNodeFromHostPort(host, port, auth)
	if err != nil {
		log.Print("Unable to obtain connection . Err:", err)
		return
	}
	c.NodeMap[name] = dnode
	node = dnode
	return node, nil
}

// GetPodAuth will return an authstring from the local config and/or the
// groupcache group if it is in there. This probably still needs a bit of work
// to be reliable enough for me.
func (c *Constellation) GetPodAuth(podname string) string {
	return c.AuthCache.Get(podname)
}

// StartCache is used to start up the groupcache mechanism
func (c *Constellation) StartCache() {
	log.Print("Starting AuthCache")
	var peers []string
	if c.PeerList == nil {
		log.Print("Initializing PeerList")
		c.PeerList = make(map[string]string)
	}

	for _, peer := range c.PeerList {
		log.Printf("Assigning peer '%s'", peer)
		if peer > "" {
			peers = append(peers, "http://"+peer+":"+GCPORT)
		}
	}
	c.Peers.Set(peers...)
	var authcache = groupcache.NewGroup(c.Groupname, 64<<20, groupcache.GetterFunc(
		func(ctx groupcache.Context, key string, dest groupcache.Sink) error {
			pod := key
			auth := c.GetAuthForPodFromConfig(pod)
			if auth > "" {
				dest.SetString(auth)
				return nil
			}
			err := fmt.Errorf("Found no value for auth on pod %s, not storing anything", pod)
			return err
		}))
	c.AuthCache = NewCache(authcache)

}

// LoadLocalPods uses the PodConfigs read from the sentinel config file and
// talks to the local sentinel to develop the list of pods the local sentinel
// knows about.
func (c *Constellation) LoadLocalPods() error {
	if c.LocalPodMap == nil {
		c.LocalPodMap = make(map[string]*RedisPod)
	}
	if c.RemotePodMap == nil {
		c.RemotePodMap = make(map[string]*RedisPod)
	}
	// Initialize local sentinel
	if c.LocalSentinel.Name == "" {
		log.Print("Initializing LOCAL sentinel")
		var address string
		var err error
		if c.SentinelConfig.Host == "" {
			log.Print("No Hostname, determining local hostname")
			myhostname, err := os.Hostname()
			if err != nil {
				log.Print(err)
			}
			myip, err := net.LookupHost(myhostname)
			if err != nil {
				log.Fatal(err)
			}
			c.LocalSentinel.Host = myip[0]
			c.SentinelConfig.Host = myip[0]
			log.Printf("%+v", myip)
			address = fmt.Sprintf("%s:%d", myip[0], c.SentinelConfig.Port)
			c.LocalSentinel.Name = address
			log.Printf("Determined LOCAL address is: %s", address)
			log.Printf("Determined LOCAL name is: %s", c.LocalSentinel.Name)
		} else {
			address = fmt.Sprintf("%s:%d", c.SentinelConfig.Host, c.SentinelConfig.Port)
			log.Printf("Determined LOCAL address is: %s", address)
			c.LocalSentinel.Name = address
			log.Printf("Determined LOCAL name is: %s", c.LocalSentinel.Name)
		}
		c.LocalSentinel.Host = c.SentinelConfig.Host
		c.LocalSentinel.Port = c.SentinelConfig.Port
		c.LocalSentinel.Connection, err = client.DialWithConfig(&client.DialConfig{Address: address})
		if err != nil {
			// Handle error reporting here!
			//log.Printf("SentinelConfig=%+v", c.SentinelConfig)
			log.Fatalf("LOCAL Sentinel '%s' failed connection attempt", c.LocalSentinel.Name)
		}
		c.LocalSentinel.Info, _ = c.LocalSentinel.Connection.SentinelInfo()
	}
	for pname, pconfig := range c.SentinelConfig.ManagedPodConfigs {
		mi, err := c.LocalSentinel.GetMaster(pname)
		if err != nil {
			log.Printf("WARNING: Pod '%s' in config but not found when talking to the sentinel controller. Err: '%s'", pname, err)
			continue
		}
		address := fmt.Sprintf("%s:%d", mi.Host, mi.Port)
		_, err = c.GetNode(address, pname, pconfig.AuthToken)
		if err != nil {
			log.Printf("Was unable to get node '%s' for pod '%s' with auth '%s'", address, pname, pconfig.AuthToken)
		}
		pod, err := c.LocalSentinel.GetPod(pname)
		if err != nil {
			log.Printf("ERROR: No pod found on LOCAL sentinel for %s", pname)
		}
		c.LocalPodMap[pod.Name] = &pod
		c.LoadNodesForPod(&pod, &c.LocalSentinel)
	}

	log.Print("Done with LocalSentinel initialization")
	return nil
}

// GetAuthForPodFromConfig looks in the local sentinel config file to find an
// authentication token for the given pod.
func (c *Constellation) GetAuthForPodFromConfig(podname string) string {
	config, _ := c.SentinelConfig.ManagedPodConfigs[podname]
	return config.AuthToken
}

// IsBalanced is likely to be deprecated. What it currently does is to look
// across the known sentinels and pods and determine if any pod is
// "unbalanced".
func (c *Constellation) IsBalanced() (isbal bool) {
	if c.Balanced == false {
		return c.Balanced
	}
	isbal = true
	needed_monitors := 0
	monitors := 0
	for _, sentinel := range c.GetAllSentinelsQuietly() {
		pc := sentinel.PodCount()
		monitors += pc
	}
	for name, sentinels := range c.PodToSentinelsMap {
		// First try to get from local sentinel, then iterate over the rest to
		// find it
		pod, err := c.LocalSentinel.GetPod(name)
		if err != nil {
			for _, s := range sentinels {
				pod, err = s.GetPod(name)
				if err == nil {
					break
				}
			}
		}
		if pod.Name == "" {
			log.Printf("WARNING: Unable to get pod '%s' from anywhere", name)
		}
		needed := pod.Info.Quorum + 1
		needed_monitors += needed
		if len(sentinels) == 0 {
			log.Printf("WARNING: Pod %s has no sentinels?? trying to find some", pod.Name)
			sentinels = c.GetSentinelsForPod(pod.Name)
			c.PodToSentinelsMap[pod.Name] = sentinels
		}
		pod.SentinelCount = len(sentinels)
		if pod.SentinelCount < needed {
			log.Printf("Pod '%s' has %d of %d needed sentinels monitoring it, thus we are unbalanced", pod.Name, pod.SentinelCount, needed)
			isbal = false
			c.Balanced = isbal
			return isbal
		}
	}
	if needed_monitors > monitors {
		log.Printf("Need total of %d monitors, have %d", needed_monitors, monitors)
		isbal = false
	}
	c.Balanced = isbal
	return isbal
}

// MonitorPod is used to add a pod/master to the constellation cluster.
func (c *Constellation) MonitorPod(podname, address string, port, quorum int, auth string) (ok bool, err error) {
	_, havekey := c.LocalPodMap[podname]
	if havekey {
		err = fmt.Errorf("C:MP -> Pod '%s' already being monitored", podname)
		return false, err
	}
	_, havekey = c.RemotePodMap[podname]
	if havekey {
		err = fmt.Errorf("C:MP -> Pod '%s' already being monitored", podname)
		return false, err
	}
	quorumReached := false
	successfulSentinels := 0
	var pod RedisPod
	neededSentinels := quorum + 1

	sentinels, err := c.GetAvailableSentinels(podname, neededSentinels)
	if err != nil {
		log.Print("NO sentinels available! Error:", err)
		return false, err
	}
	c.PodAuthMap[podname] = auth
	cfg := SentinelPodConfig{Name: podname, AuthToken: auth, IP: address, Port: port, Quorum: quorum}
	c.SentinelConfig.ManagedPodConfigs[podname] = cfg
	isLocal := false
	for _, sentinel := range sentinels {
		log.Printf("C:MP -> Adding pod to %s", sentinel.Name)
		if sentinel.Name == c.LocalSentinel.Name {
			isLocal = true
		}
		pod, err = sentinel.MonitorPod(podname, address, port, quorum, auth)
		successfulSentinels++
	}
	// I generally dislike sleeps. Hoeever in
	// this case it is a decent ooption for refreshing data from the
	// sentinels
	time.Sleep(2 * time.Second)
	pod.SentinelCount = successfulSentinels
	if isLocal {
		c.LocalPodMap[podname] = &pod
	} else {
		c.RemotePodMap[podname] = &pod
	}
	c.PodToSentinelsMap[podname] = sentinels
	quorumReached = successfulSentinels >= quorum
	if !quorumReached {
		return false, fmt.Errorf("C:MP -> Quorum not reached for pod '%s'", podname)
	}
	return true, nil
}

// RemovePod removes a pod from each of it's sentinels.
// TODO: It neds removed from the sentinel's and the constellations'
// mappings as well
func (c *Constellation) RemovePod(podname string) (ok bool, err error) {
	sentinels, _ := c.GetAllSentinels()
	for _, sentinel := range sentinels {
		log.Printf("Removing pod from %s", sentinel.Name)
		ok, err := sentinel.RemovePod(podname)
		if err == nil && ok {
			return ok, err
		}
	}
	return ok, err
}

// Initiates a failover on a given pod.
func (c *Constellation) Failover(podname string) (ok bool, err error) {
	// change this to iterate over known sentinels via the
	// GetSentinelsForPod call
	didFailover := false
	for _, s := range c.PodToSentinelsMap[podname] {
		didFailover, err = s.DoFailover(podname)
		if didFailover {
			return true, nil
		}
	}
	return didFailover, err
}

// GetAllSentinels returns all known sentinels
func (c *Constellation) GetAllSentinels() (sentinels []*Sentinel, err error) {
	for name, pod := range c.LocalPodMap {
		slist, _ := c.LocalSentinel.GetSentinels(name)
		for _, sent := range slist {
			if sent.Name == c.LocalSentinel.Name {
				continue
			}
			_, exists := c.RemoteSentinels[sent.Name]
			if !exists {
				c.AddSentinelByAddress(sent.Name)
				log.Printf("Added REMOTE sentinel '%s' for LOCAL pod", sent.Name)
			}
		}
		c.PodToSentinelsMap[name] = slist
		pod.SentinelCount = len(slist)
		c.LocalPodMap[pod.Name] = pod
	}
	for name, pod := range c.RemotePodMap {
		_, islocal := c.LocalPodMap[pod.Name]
		if islocal {
			continue
		}
		slist, _ := c.LocalSentinel.GetSentinels(name)
		for _, sent := range slist {
			if sent.Name == c.LocalSentinel.Name {
				continue
			}
			_, exists := c.RemoteSentinels[sent.Name]
			if !exists {
				c.RemoteSentinels[sent.Name] = sent
				c.AddSentinelByAddress(sent.Name)
				log.Printf("Added REMOTE sentinel '%s' for REMOTE pod", sent.Name)
			}
		}
		pod.SentinelCount = len(slist)
		c.RemotePodMap[pod.Name] = pod
	}
	for _, s := range c.RemoteSentinels {
		_, err := s.GetPods()
		if err != nil {
			log.Printf("Sentinel %s -> GetPods err: '%s'", s.Name, err)
			continue
		}
		sentinels = append(sentinels, s)
	}
	sentinels = append(sentinels, &c.LocalSentinel)
	return sentinels, nil
}

// GetSentinelsForPod returns all sentinels the pod is monitored by. In
// other words, the pod's constellation
func (c *Constellation) GetSentinelsForPod(podname string) (sentinels []*Sentinel) {
	pod, err := c.GetPod(podname)
	if err != nil || pod == nil {
		log.Printf("Unable to get pod '%s' from constellation", podname)
		return
	}
	all_sentinels, _ := c.GetAllSentinels()
	knownSentinels := make(map[string]*Sentinel)
	var current_sentinels []*Sentinel
	for _, s := range all_sentinels {
		conn, err := s.GetConnection()
		if err != nil {
			log.Printf("Unable to connect to sentinel %s", s.Name)
			continue
		}
		defer conn.ClosePool()
		reportedSentinels, _ := conn.SentinelSentinels(podname)
		if len(reportedSentinels) == 0 {
			log.Printf("Sentinel %s was reported as having pod %s. It doesn't. Pod Needs Reset", s.Name, podname)
			continue
		}
		slist, err := s.GetSentinels(podname)
		if err != nil {
			log.Print(err)
			continue
		}
		knownSentinels[s.Name] = s

		// deal with Sentinel not having updated info on sentinels for example
		// if a sentinel loses a pod, nothing is updated. we need to catch
		// this.
		if len(slist) > 0 {
			for _, sentinel := range slist {
				_, known := knownSentinels[sentinel.Name]
				if known {
					continue
				}
				p, err := sentinel.GetSentinels(podname)
				if err != nil {
					log.Printf("Sentinel %s was reported as having pod %s. It doesn't. Pod Needs Reset", sentinel.Name, podname)
					log.Print("GetPod Err:", err)
				} else {
					if len(p) == 0 {
						log.Printf("Sentinel %s was reported as having pod %s. It doesn't. Pod Needs Reset", sentinel.Name, podname)
						continue
					}
					knownSentinels[sentinel.Name] = sentinel
				}
			}
		}
	}
	for _, sentinel := range knownSentinels {
		current_sentinels = append(current_sentinels, sentinel)
	}
	c.PodToSentinelsMap[podname] = current_sentinels
	pod.SentinelCount = len(current_sentinels)
	log.Printf("Found %d known sentinels for pod %s", pod.SentinelCount, pod.Name)
	return current_sentinels
}

// GetAvailableSentinels returns a list of sentinels the give pod is *not*
// already monitored by. It will return the least-used of the available
// sentinels in an effort to level sentinel use.
func (c *Constellation) GetAvailableSentinels(podname string, needed int) (sentinels []*Sentinel, err error) {
	all, err := c.GetAllSentinels()
	if err != nil {
		return sentinels, err
	}
	if len(all) < needed {
		return sentinels, fmt.Errorf("Not enough sentinels to achieve quorum!")
	}
	pcount := func(s1, s2 *Sentinel) bool { return s1.PodCount() < s2.PodCount() }
	By(pcount).Sort(all)
	if len(all) < needed {
		log.Fatalf("WTF? needed %d sentinels but only %d available?", needed, len(sentinels))
	}
	// time to do some tricky testing to ensure we get valid sentinels: ones
	// which do not already have this pod on them
	// THis might be cleaner with a dl-list
	w := 0 // write index
	existing_sentinels := c.GetSentinelsForPod(podname)
	if len(existing_sentinels) > 0 {
	loop:
		for _, x := range all {
			for _, es := range existing_sentinels {
				if es.Name == x.Name {
					continue loop
				}
			}
			all[w] = x
			w++
		}
	}
	usable := all[:needed]
	return usable, nil
}

// AddSentinelByAddress is a convenience function to add a sentinel by
// it's ip:port string
func (c *Constellation) AddSentinelByAddress(address string) error {
	apair := strings.Split(address, ":")
	ip := apair[0]
	port, _ := strconv.Atoi(apair[1])
	return c.AddSentinel(ip, port)
}

// SetPeers is used when the peers list for groupcache may have changed
func (c *Constellation) SetPeers() error {
	var peers []string
	for _, peer := range c.PeerList {
		if peer > "" {
			peers = append(peers, "http://"+peer+":"+GCPORT)
		}
	}
	c.Peers.Set(peers...)
	return nil
}

// LoadRemoteSentinels interrogates all known remote sentinels and crawls
// the results to explore non-local configuration
func (c *Constellation) LoadRemoteSentinels() {
	for k := range c.ConfiguredSentinels {
		log.Printf("INIT REMOTE SENTINEL: %s", k)
		c.AddSentinelByAddress(k)
	}
}

// AddSentinel adds a sentinel to the constellation
func (c *Constellation) AddSentinel(ip string, port int) error {
	if c.LocalSentinel.Name == "" {
		log.Print("Initializing LOCAL sentinel")
		if c.SentinelConfig.Host == "" {
			myhostname, err := os.Hostname()
			if err != nil {
				log.Print(err)
			}
			myip, err := net.LookupHost(myhostname)
			if err != nil {
				log.Print(err)
			}
			c.LocalSentinel.Host = myip[0]
		}
		address := fmt.Sprintf("%s:%d", c.SentinelConfig.Host, c.SentinelConfig.Port)
		c.LocalSentinel.Name = address
		var err error
		c.LocalSentinel.Connection, err = client.DialWithConfig(&client.DialConfig{Address: address})
		if err != nil {
			// Handle error reporting here! I don't thnk we want to do a
			// fatal here anymore
			log.Fatalf("LOCAL Sentinel '%s' failed connection attempt", c.LocalSentinel.Name)
		}
		c.LocalSentinel.Info, _ = c.LocalSentinel.Connection.SentinelInfo()
	}
	var sentinel Sentinel
	if port == 0 {
		err := fmt.Errorf("AddSentinel called w/ZERO port .. wtf, man?")
		return err
	}
	address := fmt.Sprintf("%s:%d", ip, port)
	log.Printf("*****************] Local Name: %s Add Called For: %s", c.LocalSentinel.Name, address)
	if address == c.LocalSentinel.Name {
		return nil
	}
	_, exists := c.RemoteSentinels[address]
	if exists {
		return nil
	}
	_, exists = c.BadSentinels[address]
	if exists {
		return nil
	}
	// Now to add to the PeerList for GroupCache
	// For now we are using just the IP and expect port 8000 by convention
	// This will change to serf/consul when that part is added I expect
	if c.PeerList == nil {
		c.PeerList = make(map[string]string)
	}
	_, exists = c.PeerList[address]
	if !exists {
		log.Print("New Peer: ", address)
		c.PeerList[address] = ip
		c.SetPeers()
	}
	c.PeerList[address] = ip
	sentinel.Name = address
	sentinel.Host = ip
	sentinel.Port = port
	_, known := c.RemoteSentinels[address]
	if known {
		log.Printf("Already have crawled '%s'", sentinel.Name)
	} else {
		log.Printf("Adding REMOTE Sentinel '%s'", address)
		conn, err := client.DialWithConfig(&client.DialConfig{Address: address})
		if err != nil {
			// Handle error reporting here!
			err = fmt.Errorf("AddSentinel -> '%s' failed connection attempt", address)
			c.BadSentinels[address] = &sentinel
			return err
		}
		sentinel.Connection = conn
		sentinel.Info, _ = sentinel.Connection.SentinelInfo()
		if address != c.LocalSentinel.Name {
			log.Print("discovering pods on remote sentinel " + sentinel.Name)
			sentinel.LoadPods()
			pods, _ := sentinel.GetPods()
			log.Printf(" %d Pods to load from %s ", len(pods), address)
			c.RemoteSentinels[address] = &sentinel
			for _, pod := range pods {
				if pod.Name == "" {
					log.Print("WUT: Have a nameless pod. This is probably a bug.")
					continue
				}
				_, islocal := c.LocalPodMap[pod.Name]
				if islocal {
					log.Print("Skipping local pod")
					continue
				}
				_, isremote := c.RemotePodMap[pod.Name]
				if isremote {
					log.Print("Skipping known remote pod")
					continue
				}
				log.Print("Adding DISCOVERED remotely managed pod " + pod.Name)
				c.GetPodAuth(pod.Name)
				log.Print("Got auth for pod")
				c.LoadNodesForPod(&pod, &sentinel)
				newsentinels, _ := sentinel.GetSentinels(pod.Name)
				pod.SentinelCount = len(newsentinels)
				c.PodToSentinelsMap[pod.Name] = newsentinels
				c.RemotePodMap[pod.Name] = &pod
				for _, ns := range newsentinels {
					_, known := c.RemoteSentinels[ns.Name]
					if known {
						continue
					}
					if ns.Name == c.LocalSentinel.Name || ns.Name == sentinel.Name {
						continue
					}
					c.AddSentinelByAddress(ns.Name)
				}
			}
		}
	}
	return nil
}

// LoadNodesForPod is called to add the master and slave nodes for the
// given pod.
func (c *Constellation) LoadNodesForPod(pod *RedisPod, sentinel *Sentinel) {
	mi, err := sentinel.GetMaster(pod.Name)
	if err != nil {
		log.Printf("WARNING: Pod '%s' in config but not found when talking to the sentinel controller. Err: '%s'", pod.Name, err)
		return
	}
	address := fmt.Sprintf("%s:%d", mi.Host, mi.Port)
	node, err := c.GetNode(address, pod.Name, pod.AuthToken)
	if err != nil {
		log.Printf("Was unable to get node '%s' for pod '%s' with auth '%s'", address, pod.Name, pod.AuthToken)
		pod.ValidAuth = false
		return
	}
	slaves := node.Slaves
	for _, si := range slaves {
		c.GetNode(si.Name, pod.Name, pod.AuthToken)
	}

}

// GetAllSentinelsQuietly is a convenience function used primarily in the
// UI.
func (c *Constellation) GetAllSentinelsQuietly() (sentinels []*Sentinel) {
	sentinels, _ = c.GetAllSentinels()
	return
}

// SentinelCount returns the number of known sentinels, including this one
func (c *Constellation) SentinelCount() int {
	return len(c.RemoteSentinels) + 1
}

// LoadRemotePods loads pods discovered through remote sentinel
// interrogation or througg known-sentinel directives
func (c *Constellation) LoadRemotePods() error {
	if c.RemotePodMap == nil {
		c.RemotePodMap = make(map[string]*RedisPod)
	}
	sentinels := []*Sentinel{&c.LocalSentinel}
	log.Printf("Loading pods on %d sentinels", len(sentinels))
	if len(sentinels) == 0 {
		err := fmt.Errorf("C:LP-> ERROR: All Sentinels failed connection") // This error is becoming more common in the code perhaps moving it to a dedicated method?
		log.Println(err)
		return err
	}
	for _, sentinel := range c.RemoteSentinels {
		if sentinel.Name != c.LocalSentinel.Name {
			pods, err := sentinel.GetPods()
			if err != nil {
				log.Print("C:LP-> sentinel error:", err)
				continue
			}
			for _, pod := range pods {
				if pod.Name == "" {
					log.Print("WUT: Have a nameless pod. This is probably a bug.")
					continue
				}
				_, islocal := c.LocalPodMap[pod.Name]
				if islocal {
					continue
				}
				_, err := sentinel.GetSentinels(pod.Name)
				if err != nil {
					log.Printf("WTF? Sentinel returned no sentinels list for it's own pod '%s'", pod.Name)
				} else {
					podauth := c.GetPodAuth(pod.Name)
					pod.AuthToken = podauth
					c.RemotePodMap[pod.Name] = &pod
				}
			}
		}
	}
	return nil
}

// GetAnySentinel is deprecated and calls to it need to be found and
// refactored so it can die.
func (c *Constellation) GetAnySentinel() (sentinel Sentinel, err error) {
	// randomized code pulled out. this needs to just go away.
	return c.LocalSentinel, nil
}

// HasPodsInErrorState returns true if at least one pod is in an error
// state.
// TODO: this needs to be "cloned" to a HasPodsInWarningState when that
// refoctoring takes place.
func (c *Constellation) HasPodsInErrorState() bool {
	if c.ErrorPodCount() > 0 {
		return true
	}
	return false
}

// ErrorPodCount returns the number of pods currently reporting errors
func (c *Constellation) ErrorPodCount() (count int) {
	var epods []*RedisPod
	errormap := make(map[string]*RedisPod)
	for _, pod := range c.LocalPodMap {
		if pod.HasErrors() {
			errormap[pod.Name] = pod
		}
	}
	for _, pod := range c.RemotePodMap {
		_, islocal := c.LocalPodMap[pod.Name]
		if !islocal {
			if pod.HasErrors() {
				errormap[pod.Name] = pod
			}
		}
	}
	for _, pod := range errormap {
		epods = append(epods, pod)
	}
	c.PodsInError = epods
	return len(epods)
}

// GetPodsInError is used to get the list of pods currently reporting
// errors
func (c *Constellation) GetPodsInError() (errors []*RedisPod) {
	c.ErrorPodCount()
	return c.PodsInError
}

// PodCount updates current pod information and returns the number of pods
// managed by the constellation
func (c *Constellation) PodCount() int {
	podmap, _ := c.GetPodMap()
	return len(podmap)
}

// BalancePod is used to rebalance a pod. This means pulling a lis tof
// available sentinels, determining how many are "missing" and adding
// the pod to the appropriate number of sentinels to bring it up to spec
func (c *Constellation) BalancePod(pod *RedisPod) {
	pod, _ = c.GetPod(pod.Name) // testing a theory
	log.Print("Balance called on Pod" + pod.Name)
	neededTotal := pod.Info.Quorum + 1
	sentinels := c.GetSentinelsForPod(pod.Name)
	pod.SentinelCount = len(sentinels)
	log.Printf("Pod needs %d sentinels, has %d sentinels", neededTotal, pod.SentinelCount)
	if pod.SentinelCount < neededTotal {
		log.Printf("Attempting rebalance of %s \n'%+v' ", pod.Name, pod)
		needed := neededTotal - pod.SentinelCount
		if pod.AuthToken == "" {
			pod.AuthToken = c.GetPodAuth(pod.Name)
		}
		log.Printf("%s on %d sentinels, needs %d more", pod.Name, pod.SentinelCount, needed)
		sentinels, _ := c.GetAvailableSentinels(pod.Name, needed)
		log.Printf("Request %d sentinels for %s, got %d to use", needed, pod.Name, len(sentinels))
		isLocal := false
		for _, sentinel := range sentinels {
			log.Print("Adding to sentinel ", sentinel.Name)
			if sentinel.Name == c.LocalSentinel.Name {
				isLocal = true
			}
			pod, err := sentinel.MonitorPod(pod.Name, pod.Info.IP, pod.Info.Port, pod.Info.Quorum, pod.AuthToken)
			if err != nil {
				log.Printf("Sentinel %s Pod: %s, Error: %s", sentinel.Name, pod.Name, err)
				continue
			}
			c.PodToSentinelsMap[pod.Name] = append(c.PodToSentinelsMap[pod.Name], sentinel)
		}
		time.Sleep(500 * time.Millisecond) // wait for propagation between sentinels
		slist := c.GetSentinelsForPod(pod.Name)
		c.PodToSentinelsMap[pod.Name] = slist
		pod.SentinelCount = len(slist)
		if isLocal {
			c.LocalPodMap[pod.Name] = pod
		} else {
			c.RemotePodMap[pod.Name] = pod
		}
		log.Printf("Rebalance of %s completed, it now has %d sentinels", pod.Name, pod.SentinelCount)
	}
}

// Balance will attempt to balance the constellation
// A constellation is unbalanced if any pod is not listed as managed by enough
// sentinels to achieve quorum+1
// It will first verify the current balance state to avoid unnecessary balance
// attempts.
// This will likely be deprecated
func (c *Constellation) Balance() {
	log.Print("Balance called on constellation")
	c.HasPodsInErrorState()
	allpods := c.GetPods()

	log.Printf("Constellation rebalance initiated, have %d pods unbalanced", len(c.PodsInError))
	for _, pod := range allpods {
		c.BalancePod(pod)
	}
	c.Balanced = true
}

// Getmaster returns the current client.MasterAddress struct for the given
// pod
func (c *Constellation) GetMaster(podname string) (master client.MasterAddress, err error) {
	sentinels, _ := c.GetAllSentinels()
	for _, sentinel := range sentinels {
		master, err := sentinel.GetMaster(podname)
		if err == nil {
			return master, nil
		}
	}
	return master, fmt.Errorf("No Sentinels available for pod '%s'", podname)
}

// GetPod returns a *RedisPod instance for the given podname
func (c *Constellation) GetPod(podname string) (pod *RedisPod, err error) {
	pod, islocal := c.LocalPodMap[podname]
	if islocal {
		spod, err := c.LocalSentinel.GetPod(podname)
		address := fmt.Sprintf("%s:%d", spod.Info.IP, spod.Info.Port)
		auth := spod.AuthToken
		if auth == "" {
			auth = c.GetPodAuth(podname)
		}
		master, _ := c.GetNode(address, podname, auth)
		spod.Master = master
		c.LocalSentinel.GetSlaves(podname)
		c.LoadNodesForPod(pod, &c.LocalSentinel)
		pod = &spod
		c.LocalPodMap[podname] = pod
		return pod, err
	}
	sentinels, _ := c.GetAllSentinels()
	for _, s := range sentinels {
		conn, err := s.GetConnection()
		if err != nil {
			log.Printf("Unable to connect to sentinel '%s'", s.Name)
			continue
		}
		defer conn.ClosePool()
		mi, _ := conn.SentinelMasterInfo(podname)
		if mi.Name == podname {
			auth := c.GetPodAuth(podname)
			address := fmt.Sprintf("%s:%d", mi.IP, mi.Port)
			master, err := c.GetNode(address, podname, auth)
			if err != nil {
				log.Print("Unable to get master node ===========")
			}
			pod, _ := NewMasterFromMasterInfo(mi, auth)
			pod.Master = master
			c.RemotePodMap[podname] = &pod
			return &pod, nil
		}
	}

	if err != nil {
		log.Printf("Could NOT load pod '%s' from %s", podname, err)
		return pod, err
	}
	return pod, nil
}

// GetSlaves return a list of client.SlaveInfo structs for the given pod
func (c *Constellation) GetSlaves(podname string) (slaves []client.SlaveInfo, err error) {
	sentinels, err := c.GetAllSentinels()
	for _, sentinel := range sentinels {
		slaves, err = sentinel.GetSlaves(podname)
		if err == nil {
			return
		}
	}
	return
}

// GetPods returns the list of known pods
func (c *Constellation) GetPods() (pods []*RedisPod) {
	podmap, _ := c.GetPodMap()
	havepods := make(map[string]interface{})
	for _, pod := range podmap {
		if pod.Name == "" {
			log.Print("WUT: Have a nameless pod. Probably a bug")
			continue
		}
		_, have := havepods[pod.Name]
		if !have {
			pods = append(pods, pod)
		}
	}
	return pods
}

// GetPodMap returs the current pod mapping. This combines local and
// remote sentinels to get all known pods in the cluster
func (c *Constellation) GetPodMap() (pods map[string]*RedisPod, err error) {
	pods = make(map[string]*RedisPod)
	for k, v := range c.LocalPodMap {
		pods[k] = v
	}
	for k, v := range c.RemotePodMap {
		_, local := c.LocalPodMap[k]
		if !local {
			_, haveit := pods[k]
			if !haveit {
				pods[k] = v
			}
		}
	}
	return pods, nil
}

// extractSentinelDirective parses the sentinel directives from the
// sentinel config file
func (c *Constellation) extractSentinelDirective(entries []string) error {
	switch entries[0] {
	case "monitor":
		pname := entries[1]
		port, _ := strconv.Atoi(entries[3])
		quorum, _ := strconv.Atoi(entries[4])
		spc := SentinelPodConfig{Name: pname, IP: entries[2], Port: port, Quorum: quorum}
		spc.Sentinels = make(map[string]string)
		// normally we should not see duplicate IP:PORT combos, however it
		// can happen when people do things manually and dont' clean up.
		// We need to detect them and ignore the second one if found,
		// reporting the error condition this will require tracking
		// ip:port pairs...
		addr := fmt.Sprintf("%s:%d", entries[2], port)
		_, exists := c.SentinelConfig.ManagedPodConfigs[addr]
		if !exists {
			c.SentinelConfig.ManagedPodConfigs[entries[1]] = spc
		}
		return nil

	case "auth-pass":
		pname := entries[1]
		pc := c.SentinelConfig.ManagedPodConfigs[pname]
		pc.AuthToken = entries[2]
		c.PodAuthMap[pname] = pc.AuthToken
		c.SentinelConfig.ManagedPodConfigs[pname] = pc
		return nil

	case "known-sentinel":
		podname := entries[1]
		sentinel_address := entries[2] + ":" + entries[3]
		pc := c.SentinelConfig.ManagedPodConfigs[podname]
		pc.Sentinels[sentinel_address] = ""
		if c.Peers == nil {
			// This means the sentinel config has no bind statement
			// So we will pull the local IP and use it
			// I don't like this but dont' have a great option either.
			myhostname, err := os.Hostname()
			if err != nil {
				log.Print(err)
			}
			myip, err := net.LookupHost(myhostname)
			if err != nil {
				log.Print(err)
			}
			c.LocalSentinel.Host = myip[0]
			log.Printf("NO BIND STATEMENT FOUND. USING: '%s'", c.LocalSentinel.Host)
			me := "http://" + c.SentinelConfig.Host + ":" + GCPORT
			c.Peers = groupcache.NewHTTPPool(me)
			c.PeerList[c.SentinelConfig.Host+fmt.Sprintf(":%d", c.SentinelConfig.Port)] = c.SentinelConfig.Host
			c.SetPeers()
			go http.ListenAndServe(c.SentinelConfig.Host+":"+GCPORT, http.HandlerFunc(c.Peers.ServeHTTP))
			c.StartCache()
		}
		c.ConfiguredSentinels[sentinel_address] = sentinel_address
		return nil

	case "known-slave":
		// Currently ignoring this, but may add call to a node manager.
		return nil

	case "config-epoch", "leader-epoch", "current-epoch", "down-after-milliseconds":
		// We don't use these keys
		return nil

	default:
		err := fmt.Errorf("Unhandled sentinel directive: %+v", entries)
		log.Print(err)
		return nil
	}
}

// LoadSentinelConfigFile loads the local config file pulled from the
// environment variable "REDSKULL_SENTINELCONFIGFILE"
func (c *Constellation) LoadSentinelConfigFile() error {
	file, err := os.Open(c.SentinelConfigName)
	if err != nil {
		log.Print(err)
		return err
	}
	defer file.Close()
	bf := bufio.NewReader(file)
	for {
		rawline, err := bf.ReadString('\n')
		if err == nil || err == io.EOF {
			line := strings.TrimSpace(rawline)
			// ignore comments
			if strings.Contains(line, "#") {
				continue
			}
			entries := strings.Split(line, " ")
			//Most values are key/value pairs
			switch entries[0] {
			case "sentinel": // Have a sentinel directive
				err := c.extractSentinelDirective(entries[1:])
				if err != nil {
					// TODO: Fix this to return a different error if we can't
					// connect to the sentinel
					log.Printf("Misshapen sentinel directive: '%s'", line)
				}
			case "port":
				iport, _ := strconv.Atoi(entries[1])
				c.SentinelConfig.Port = iport
				//log.Printf("Local sentinel is bound to port %d", c.SentinelConfig.Port)
			case "dir":
				c.SentinelConfig.Dir = entries[1]
			case "bind":
				if c.LocalOverrides.BindAddress > "" {
					log.Printf("Overriding Sentinel BIND directive '%s' with '%s'", entries[1], c.LocalOverrides.BindAddress)
				} else {
					c.SentinelConfig.Host = entries[1]
					if c.Peers == nil {
						me := "http://" + c.SentinelConfig.Host + ":" + GCPORT
						c.Peers = groupcache.NewHTTPPool(me)
						c.PeerList[c.SentinelConfig.Host+fmt.Sprintf(":%d", c.SentinelConfig.Port)] = c.SentinelConfig.Host
						c.SetPeers()
						go http.ListenAndServe(c.SentinelConfig.Host+":"+GCPORT, http.HandlerFunc(c.Peers.ServeHTTP))
						c.StartCache()
					}
				}
				log.Printf("Local sentinel is listening on IP %s", c.SentinelConfig.Host)
			case "":
				if err == io.EOF {
					log.Print("File load complete?")
					if c.LocalOverrides.BindAddress > "" {
						c.SentinelConfig.Host = c.LocalOverrides.BindAddress
						log.Printf("Local sentinel is listening on IP %s", c.SentinelConfig.Host)
					} else {
						if c.Peers == nil {
							// This means the sentinel config has no bind statement
							// So we will pull the local IP and use it
							// I don't like this but dont' have a great option either.
							myhostname, err := os.Hostname()
							if err != nil {
								log.Print(err)
							}
							myip, err := net.LookupHost(myhostname)
							if err != nil {
								log.Print(err)
							}
							c.LocalSentinel.Host = myip[0]
							log.Printf("NO BIND STATEMENT FOUND. USING: '%s'", c.LocalSentinel.Host)
							me := "http://" + c.SentinelConfig.Host + ":" + GCPORT
							c.Peers = groupcache.NewHTTPPool(me)
							c.PeerList[c.SentinelConfig.Host+fmt.Sprintf(":%d", c.SentinelConfig.Port)] = c.SentinelConfig.Host
							c.SetPeers()
							go http.ListenAndServe(c.SentinelConfig.Host+":"+GCPORT, http.HandlerFunc(c.Peers.ServeHTTP))
							c.StartCache()
						}
					}
					return nil
				}
				//log.Printf("Local:config -> %+v", c.SentinelConfig)
				//log.Printf("Found %d REMOTE sentinels", len(c.RemoteSentinels))
				//return nil
			default:
				log.Printf("UNhandled Sentinel Directive: %s", line)
			}
		} else {
			log.Print("=============== LOAD FILE ERROR ===============")
			log.Fatal(err)
		}
	}
}

// ResetPod this is the constellation cluster level call to issue a reset
// against the sentinels for the given pod.
func (c *Constellation) ResetPod(podname string, simultaneous bool) {
	sentinels := c.GetSentinelsForPod(podname)
	log.Printf("Calling reset on %d sentinels for pod '%s'", len(sentinels), podname)
	if len(sentinels) == 0 {
		log.Print("ERROR: Attempt to call resre on pod with no sentinels??:" + podname)
		return
	}
	for _, sentinel := range sentinels {
		log.Print("Issuing reset for " + podname)
		if simultaneous {
			go sentinel.ResetPod(podname)
		} else {
			sentinel.ResetPod(podname)
			time.Sleep(2 * time.Second)
		}
	}
	c.GetAllSentinelsQuietly()
}

// By is a convenience type to enable sorting sentinels by their
// monitored pod count
type By func(s1, s2 *Sentinel) bool

// Sort sorts sentinels using sentinelSorter
func (by By) Sort(sentinels []*Sentinel) {
	ss := &sentinelSorter{
		sentinels: sentinels,
		by:        by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ss)
}

// sentinelSorter sorts sentinels by ther "length"
type sentinelSorter struct {
	sentinels []*Sentinel
	by        func(s1, s2 *Sentinel) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *sentinelSorter) Len() int {
	return len(s.sentinels)
}

// Swap is part of sort.Interface.
func (s *sentinelSorter) Swap(i, j int) {
	s.sentinels[i], s.sentinels[j] = s.sentinels[j], s.sentinels[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *sentinelSorter) Less(i, j int) bool {
	return s.by(s.sentinels[i], s.sentinels[j])
}

// PodAuthCache is a struct used for groupcache to propogate
// authentication informaton for a pod.
type PodAuthCache struct {
	cacheGroup *groupcache.Group
	CacheType  string
}

// NewCache creates a new PodAuthCache
func NewCache(cacheGroup *groupcache.Group) *PodAuthCache {
	cache := new(PodAuthCache)
	cache.cacheGroup = cacheGroup
	return cache
}

// GetStats returns the MainCache metrics
func (pc *PodAuthCache) GetStats() groupcache.CacheStats {
	return pc.cacheGroup.CacheStats(groupcache.MainCache)
}

// GetStats returns the HotCache metrics
func (pc *PodAuthCache) GetHotStats() groupcache.CacheStats {
	return pc.cacheGroup.CacheStats(groupcache.HotCache)
}

// Get is used by GroupCache to load a key not found in the cache
func (pc *PodAuthCache) Get(podname string) string {
	var auth string
	pc.cacheGroup.Get(nil, podname, groupcache.StringSink(&auth))
	return auth
}

// GetAddressPair is a convenience function for converting an ip and port
// into the ip:port string. Probably need to move this to the common
// package
func GetAddressPair(astring string) (host string, port int, err error) {
	apair := strings.Split(astring, ":")
	host = apair[0]
	port, err = strconv.Atoi(apair[1])
	if err != nil {
		log.Printf("Unable to convert %s to port integer!", apair[1])
	}
	return
}
