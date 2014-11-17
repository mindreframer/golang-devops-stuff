package actions

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/therealbill/libredis/client"
	"github.com/therealbill/libredis/info"
)

type Sentinel struct {
	Name           string
	Host           string
	Port           int
	Connection     *client.Redis
	Errors         int
	Info           info.RedisInfoAll
	PodMap         map[string]RedisPod
	Pods           []RedisPod
	PodsInError    []RedisPod
	KnownSentinels map[string]*Sentinel
	DialConfig     client.DialConfig
}

func (s *Sentinel) GetMasters() (master []client.MasterInfo, err error) {
	conn, err := client.Dial(s.Host, s.Port)
	if err != nil {
		return
	}
	defer conn.ClosePool()
	return conn.SentinelMasters()
}

func (s *Sentinel) PodCount() int {
	s.LoadPods()
	return len(s.PodMap)
}

func (s *Sentinel) LoadPods() error {
	var pods []RedisPod
	var epods []RedisPod
	podmap := make(map[string]RedisPod)
	if s.KnownSentinels == nil {
		s.KnownSentinels = make(map[string]*Sentinel)
	}
	masters, err := s.GetMasters()
	if err != nil {
		log.Print("S:LP-> sentinel error:", err)
		return err
	}
	//log.Printf("S:LP -> sentinel %s has %d masters to load", s.Name, len(masters))
	for _, mi := range masters {
		//log.Printf("S:LP-> (%d) sentinel %s loading master: %s", i, s.Name, mi.Name)
		auth, err := s.GetPodAuthFromConfig(mi.Name)
		if err != nil {
			log.Print("GetPodAuthFromConfig returned error ", err)
			continue
		}
		if auth == "" {
			log.Printf("Pod %s is non-local, trying to return from podmap (no auth)", mi.Name)
			pod, exists := s.PodMap[mi.Name]
			if exists {
				podmap[mi.Name] = pod
				continue
			}
			log.Printf("Pod %s is neither in local podmap for sentinel %s nor has auth", mi.Name, s.Name)
			continue
		}
		rp, err := NewMasterFromMasterInfo(mi, auth)
		if err != nil {
			//log.Printf("S:LP Unable to get rp. Err: '%s'", err.Error())
			continue
		}
		podmap[mi.Name] = rp
		// Currently a bug in redis means the sentinels counts in the info
		// commands are NOT always current. So we calculate it here
		//log.Print("S:LP -> Checking on sentinels")
		pod_sentinels, err := s.GetSentinels(rp.Name)
		if err != nil {
			log.Printf("WARNING: Sentinel '%s' is recorded as havig pod '%s' but it doesn't return. Err is '%s'", s.Name, rp.Name, err)
			continue
		}
		//log.Print("S:LP GetSentinels returned")
		for _, sentinel := range pod_sentinels {
			s.KnownSentinels[sentinel.Name] = sentinel
		}
		rp.SentinelCount = len(pod_sentinels)
		//log.Printf("Sentinel %s reports pod %s has %d sentinels", s.Name, rp.Name, rp.SentinelCount)

		//log.Printf("S:LP-> got rp=%s", rp.Name)
		if rp.HasErrors() {
			epods = append(epods, rp)
		}
		podmap[mi.Name] = rp
		pods = append(pods, rp)
	}
	s.Pods = pods
	s.PodMap = podmap
	s.PodsInError = epods
	//log.Print("S:LP -> LoadPods Complete")
	return err
}

func (s *Sentinel) DoFailover(podname string) (ok bool, err error) {
	// Q: Move error handling/reporting to constellation?
	conn, err := client.Dial(s.Host, s.Port)
	if err != nil {
		return
	}
	didFailover, err := conn.SentinelFailover(podname)
	return didFailover, err
}

func (s *Sentinel) ResetPod(podname string) {
	conn, err := client.Dial(s.Host, s.Port)
	if err != nil {
		return
	}
	defer conn.ClosePool()
	err = conn.SentinelReset(podname)
	if err != nil {
		log.Print("Error on reset call for " + podname + " Err=" + err.Error())
	}
}

func (s *Sentinel) GetSlaves(podname string) (slaves []client.SlaveInfo, err error) {
	// TODO: Bubble errors to out custom error package
	// See DoFailover for an example
	conn, err := client.Dial(s.Host, s.Port)
	if err != nil {
		return
	}
	defer conn.ClosePool()
	return conn.SentinelSlaves(podname)
}

func (s *Sentinel) GetSentinels(podname string) (sentinels []*Sentinel, err error) {
	conn, err := client.Dial(s.Host, s.Port)
	if err != nil {
		return
	}
	defer conn.ClosePool()
	sinfos, err := conn.SentinelSentinels(podname)
	if err != nil {
		return sentinels, err
	}
	stracker := make(map[string]*Sentinel)
	for _, sent := range sinfos {
		sentinel := Sentinel{Name: sent.Name, Host: sent.IP, Port: sent.Port}
		conn, err := client.Dial(sent.IP, sent.Port)
		if err != nil {
			//log.Printf("Unable to connect to sentinel %s. Error reported as '%s'", sent.Name, err.Error())
			continue
		}
		defer conn.ClosePool()
		sentinel.Connection = conn
		pm, err := sentinel.Connection.SentinelGetMaster(podname)
		if err != nil || pm.Port == 0 {
			//log.Printf("S:GS -> %s said %s had pod %s but it didn't.", s.Name, sentinel.Name, podname)
		} else {
			stracker[sentinel.Name] = &sentinel
		}
	}
	for _, v := range stracker {
		sentinels = append(sentinels, v)
	}
	sentinels = append(sentinels, s)
	return sentinels, nil
}

func (s *Sentinel) GetConnection() (conn *client.Redis, err error) {
	conn, err = client.Dial(s.Host, s.Port)
	return
}

func (s *Sentinel) GetMaster(podname string) (master client.MasterAddress, err error) {
	if s.Connection == nil {
		log.Fatal("s.Connection is nil, connection not initialzed!")
	}
	conn, err := client.Dial(s.Host, s.Port)
	if err != nil {
		return
	}
	master, err = conn.SentinelGetMaster(podname)
	// TODO: THis needs changed to our custom errors package
	return
}

func (s *Sentinel) MonitorPod(podname, address string, port, quorum int, auth string) (rp RedisPod, err error) {
	// TODO: Update to new common and error packages
	//log.Printf("S:MP-> add called for %s-> %s:%d", podname, address, port)
	conn, err := client.Dial(s.Host, s.Port)
	if err != nil {
		return
	}
	_, err = conn.SentinelMonitor(podname, address, port, quorum)
	if err != nil {
		return rp, err
	}
	if auth > "" {
		ok := conn.SentinelSetString(podname, "auth-pass", auth)
		log.Print(ok)
	}
	s.LoadPods()
	rp, err = s.GetPod(podname)
	if err != nil {
		log.Printf("S:MP Error on s.GetPod: %s", err.Error())
	}
	_, err = LoadNodeFromHostPort(address, port, auth)
	if err != nil {
		return rp, fmt.Errorf("S:MP-> unable to load new pod's master node: Error: %s", err)
	}
	//log.Printf("A:S:MP-> Loaded master node")
	return rp, nil
}

func (s *Sentinel) RemovePod(podname string) (ok bool, err error) {
	conn, err := client.Dial(s.Host, s.Port)
	if err != nil {
		return
	}
	_, err = conn.SentinelRemove(podname)
	if err != nil {
		// convert to custom errors package
		return false, err
	}
	return true, err
}

func (s *Sentinel) GetPods() (pods map[string]RedisPod, err error) {
	//err = s.LoadPods()
	return s.PodMap, err
}

func (s *Sentinel) GetPod(podname string) (rp RedisPod, err error) {
	//log.Printf("Sentinel.Getpod called for pod '%s'", podname)
	conn, err := client.Dial(s.Host, s.Port)
	if err != nil {
		return
	}
	mi, err := conn.SentinelMasterInfo(podname)
	if err != nil {
		log.Print("S:GetPod failed to get master info. Err:", err)
		return rp, err
	}
	if mi.Port == 0 {
		err = fmt.Errorf("WTF?! Got nothing back from SentinelMasterInfo, not even an error")
		return rp, err
	}
	auth, _ := s.GetPodAuthFromConfig(podname)
	//log.Printf("%s :: s.GetPodAuthFromConfig = '%s'", s.Name, auth)
	rp, err = NewMasterFromMasterInfo(mi, auth)
	if auth == "" {
		err = fmt.Errorf("NO AUTH FOR POD '%s'!", mi.Name)
		return rp, err
	}
	if err != nil {
		log.Print("S:GetPod failed to get pod from master info. Err:", err)
		return rp, err
	}
	if s.PodMap == nil {
		s.PodMap = make(map[string]RedisPod)
	}
	s.PodMap[podname] = rp
	return rp, nil
}

// GetPodAuthFromConfig is a bit of a hack. It parses the sentinel config file
// looking for pods with an authtoken.
func (s *Sentinel) GetPodAuthFromConfig(podname string) (string, error) {
	if s.Info.Server.ConfigFile == "" {
		return "", nil
	}
	file, err := os.Open(s.Info.Server.ConfigFile)
	if err != nil {
		log.Printf("unable to open '%s'. Err:%s", s.Info.Server.ConfigFile, err.Error())
		return "", err
	}
	defer file.Close()
	//log.Printf("GetPodAuthFromConfig called, cfg file=%s", s.Info.Server.ConfigFile)
	bf := bufio.NewReader(file)
	var lines []string
	authmap := make(map[string]string)
ReadFile:
	for {
		//log.Print("reading line")
		line, err := bf.ReadString('\n')
		////log.Printf("Read line: '%s'", line)

		switch err {

		case nil:
			// We only care about auth-pass lines right now
			if strings.Contains(line, "sentinel auth-pass") {
				lines = append(lines, line)
			} else {
				continue
			}
		case io.EOF:
			// last line of file missing \n, but could still valid
			if strings.Contains(line, "sentinel auth-pass") {
				lines = append(lines, line)
				break ReadFile
			}
			break ReadFile
		default:
			log.Fatal(err)
		}
	}
	for _, line := range lines {
		dsplit := strings.Split(line, " ")
		name := dsplit[2]
		auth := strings.TrimRight(dsplit[3], "\n")
		authmap[name] = auth
		if name == podname {
			return auth, nil
		}
	}
	return "", nil
}
